package state_db_rocksdb

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"runtime"
	"strconv"
	"sync"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/goroutines"
	"github.com/linxGnu/grocksdb"
)

type DB struct {
	db                        *grocksdb.DB
	cf_handle_default         *grocksdb.ColumnFamilyHandle
	cf_handles                [col_COUNT]*grocksdb.ColumnFamilyHandle
	opts_r                    *grocksdb.ReadOptions
	opts_r_itr                *grocksdb.ReadOptions
	versioned_read_pools      [col_COUNT]*util.Pool
	latest_state              LatestState
	maintenance_task_executor goroutines.GoroutineGroup
	opts                      Opts
	db_opts                   *grocksdb.Options
}

const (
	col_main_trie_value_latest = iota + state_db.COL_COUNT
	col_acc_trie_value_latest
	col_config_changes
	col_COUNT
)

const (
	// Size limit of the memory structures for prune
	db_prune_buffer_max_size = 500000
)

var versioned_read_columns = []state_db.Column{state_db.COL_acc_trie_value, state_db.COL_main_trie_value}

type Opts = struct {
	Path                            string
	DisableMostRecentTrieValueViews bool
}

func (self *DB) Init(opts Opts) *DB {
	self.opts = opts
	new_db_opts := func() *grocksdb.Options {
		ret := grocksdb.NewDefaultOptions()
		ret.SetErrorIfExists(false)
		ret.SetCreateIfMissing(true)
		ret.SetCompression(grocksdb.LZ4Compression)
		ret.SetCreateIfMissingColumnFamilies(true)
		ret.IncreaseParallelism(runtime.NumCPU())
		ret.SetMaxFileOpeningThreads(runtime.NumCPU())
		ret.SetMaxBackgroundCompactions(runtime.NumCPU())
		ret.SetMaxBackgroundFlushes(runtime.NumCPU())
		ret.SetMaxOpenFiles(128) //Maybe even less
		//ret.SetEnablePipelinedWrite(true)
		//ret.SetUseAdaptiveMutex(true)
		return ret
	}
	self.db_opts = new_db_opts()
	const real_col_cnt = 1 + col_COUNT
	cf_opts_default := grocksdb.NewDefaultOptions()
	defer cf_opts_default.Destroy()
	cfnames, cfopts := [real_col_cnt]string{"default"}, [real_col_cnt]*grocksdb.Options{cf_opts_default}
	for i := state_db.Column(1); i < real_col_cnt; i++ {
		cf_opts := new_db_opts()
		defer cf_opts.Destroy()
		if col := i - 1; col == col_main_trie_value_latest || col == col_acc_trie_value_latest {
			cf_opts.SetAllowConcurrentMemtableWrites(false)
			cf_opts.OptimizeForPointLookup(300)
		}
		cfnames[i], cfopts[i] = strconv.Itoa(int(i)), cf_opts
	}
	db_opts := new_db_opts()
	defer db_opts.Destroy()
	db, cf_handles, err := grocksdb.OpenDbColumnFamilies(db_opts, opts.Path, cfnames[:], cfopts[:])
	util.PanicIfNotNil(err)
	self.db = db
	self.cf_handle_default = cf_handles[0]
	copy(self.cf_handles[:], cf_handles[1:])
	self.opts_r = func() *grocksdb.ReadOptions {
		ret := grocksdb.NewDefaultReadOptions()
		ret.SetVerifyChecksums(false)
		return ret
	}()
	self.opts_r_itr = func() *grocksdb.ReadOptions {
		ret := grocksdb.NewDefaultReadOptions()
		ret.SetVerifyChecksums(false)
		ret.SetPrefixSameAsStart(true)
		ret.SetFillCache(false)
		return ret
	}()
	self.maintenance_task_executor.Init(2, 1024) // 8KB
	for _, col := range versioned_read_columns {
		col := col
		self.versioned_read_pools[col] = new(util.Pool).Init(uint(1.5*float64(runtime.NumCPU())), func() util.PoolItem {
			return &VersionedReadContext{
				itr: self.db.NewIteratorCF(self.opts_r_itr, self.cf_handles[col]),
			}
		})
	}
	self.latest_state.Init(self)
	return self
}

func (self *DB) Snapshot(dir string, log_size_for_flush uint64) error {
	c, err := self.db.NewCheckpoint()
	if err != nil {
		return err
	}
	return c.CreateCheckpoint(dir, log_size_for_flush)
}

func (self *DB) RecreateColumn(column state_db.Column, nodes map[common.Hash][]byte) {
	range_limit := [common.VersionedKeyLength]byte{255}
	range32 := grocksdb.Range{
		Start: make([]byte, common.HashLength),
		Limit: make([]byte, common.HashLength),
	}
	copy(range32.Limit, range_limit[0:common.HashLength])
	err := self.db.DropColumnFamily(self.cf_handles[column])
	if err != nil {
		panic("Could not delete column")
	}
	self.cf_handles[column].Destroy()
	acc_trie_handle, err2 := self.db.CreateColumnFamily(self.db_opts, strconv.Itoa(int(column+1)))
	if err2 != nil {
		panic("Could not create column")
	}
	self.cf_handles[column] = acc_trie_handle

	for node_to_keep, b := range nodes {
		self.db.PutCF(grocksdb.NewDefaultWriteOptions(), self.cf_handles[column], node_to_keep[:], b)
	}
	self.db.CompactRangeCF(self.cf_handles[column], range32)
}

func (self *DB) deleteStateValues(blk_num types.BlockNum) {
	range_limit := [common.VersionedKeyLength]byte{255}
	range40 := grocksdb.Range{
		Start: make([]byte, common.VersionedKeyLength),
		Limit: make([]byte, common.VersionedKeyLength),
	}
	copy(range40.Limit, range_limit[0:common.VersionedKeyLength])
	prev_key := make([]byte, common.VersionedKeyLength)

	//Select account storage values to prune
	set_account_storage_value_to_prune := make([][]byte, 0)
	itr := self.db.NewIteratorCF(self.opts_r_itr, self.cf_handles[state_db.COL_acc_trie_value])
	itr.SeekToFirst()
	for {
		copy(prev_key, itr.Key().Data())
		itr.Key().Free()
		itr.Next()
		if !itr.Valid() {
			break
		}
		if bytes.Compare(prev_key[0:common.HashLength], itr.Key().Data()[0:common.HashLength]) == 0 {
			ver_blk_num := binary.BigEndian.Uint64(itr.Key().Data()[common.HashLength:common.VersionedKeyLength])
			if ver_blk_num < blk_num {
				set_account_storage_value_to_prune = append(set_account_storage_value_to_prune, common.CopyBytes(prev_key))
			}
		}

		if len(set_account_storage_value_to_prune) > db_prune_buffer_max_size {
			for _, value_to_remove := range set_account_storage_value_to_prune {
				self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_acc_trie_value], value_to_remove)
			}
			set_account_storage_value_to_prune = make([][]byte, 0)
		}
	}
	itr.Close()

	for _, value_to_remove := range set_account_storage_value_to_prune {
		self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_acc_trie_value], value_to_remove)
	}
	self.db.CompactRangeCF(self.cf_handles[state_db.COL_acc_trie_value], range40)
}

func (self *DB) recreateMainTrie(state_root_to_keep *[]common.Hash, blk_num types.BlockNum) {
	current_block_state := state_db.GetBlockStateReader(self, blk_num)
	//Select nodes which are not to be deleted
	nodes_to_keep := make(map[common.Hash][]byte)
	for _, root_to_keep := range *state_root_to_keep {
		current_block_state.ForEachMainNodeHashByRoot(&root_to_keep, func(h *common.Hash, b []byte) {
			nodes_to_keep[*h] = common.CopyBytes(b)
		})
	}
	self.RecreateColumn(state_db.COL_main_trie_node, nodes_to_keep)
}

func (self *DB) deleteStateRoot(blk_num types.BlockNum) {
	range_limit := [common.VersionedKeyLength]byte{255}
	range40 := grocksdb.Range{
		Start: make([]byte, common.VersionedKeyLength),
		Limit: make([]byte, common.VersionedKeyLength),
	}
	copy(range40.Limit, range_limit[0:common.VersionedKeyLength])

	//Select main trie values to prune/remove
	set_value_to_prune := make([][]byte, 0)
	set_storage_root_to_keep := make([]common.Hash, 0)
	current_block_state := state_db.GetBlockStateReader(self, blk_num)

	//Iterate over all values and select which to keep
	itr := self.db.NewIteratorCF(self.opts_r_itr, self.cf_handles[state_db.COL_main_trie_value])
	itr.SeekToFirst()
	prev_key := make([]byte, common.VersionedKeyLength)
	for {
		copy(prev_key, itr.Key().Data())
		itr.Key().Free()
		account := state_db.DecodeAccountFromTrie(itr.Value().Data())
		itr.Value().Free()
		itr.Next()
		if !itr.Valid() {
			if account.StorageRootHash != nil {
				set_storage_root_to_keep = append(set_storage_root_to_keep, *account.StorageRootHash)
			}
			break
		}
		if bytes.Compare(prev_key[0:common.HashLength], itr.Key().Data()[0:common.HashLength]) == 0 {
			//Only prune the previous value if current value for the same account is below blk_num
			ver_blk_num := binary.BigEndian.Uint64(itr.Key().Data()[common.HashLength:common.VersionedKeyLength])
			if ver_blk_num < blk_num {
				set_value_to_prune = append(set_value_to_prune, common.CopyBytes(prev_key))
			} else {
				if account.StorageRootHash != nil {
					set_storage_root_to_keep = append(set_storage_root_to_keep, *account.StorageRootHash)
				}
			}
		} else {
			if account.StorageRootHash != nil {
				set_storage_root_to_keep = append(set_storage_root_to_keep, *account.StorageRootHash)
			}
		}

		if len(set_value_to_prune) > db_prune_buffer_max_size {
			for _, value_to_remove := range set_value_to_prune {
				self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_main_trie_value], value_to_remove)
			}
			set_value_to_prune = make([][]byte, 0)
		}
	}
	itr.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, value_to_remove := range set_value_to_prune {
			self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_main_trie_value], value_to_remove)
		}
		self.db.CompactRangeCF(self.cf_handles[state_db.COL_main_trie_value], range40)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		//Select account nodes to prune
		account_nodes_to_keep := make(map[common.Hash][]byte)
		for _, root_to_keep := range set_storage_root_to_keep {
			current_block_state.ForEachAccountNodeHashByRoot(&root_to_keep, func(h *common.Hash, b []byte) {
				account_nodes_to_keep[*h] = common.CopyBytes(b)
			})
		}
		self.RecreateColumn(state_db.COL_acc_trie_node, account_nodes_to_keep)
	}()
	wg.Wait()
}

func (self *DB) Prune(state_root_to_keep []common.Hash, blk_num types.BlockNum) {
	var wg sync.WaitGroup

	// Asynchronously delete state values
	wg.Add(1)
	go func() {
		defer wg.Done()
		self.deleteStateValues(blk_num)
	}()

	// Asynchronously recreate Main trie
	wg.Add(1)
	go func() {
		defer wg.Done()
		self.recreateMainTrie(&state_root_to_keep, blk_num)
	}()

	// Asynchronously delete state root and main trie values
	wg.Add(1)
	go func() {
		defer wg.Done()
		self.deleteStateRoot(blk_num)
	}()

	wg.Wait()
}

func (self *DB) Close() {
	self.latest_state.Close()
	self.invalidate_versioned_read_pools()
	self.maintenance_task_executor.JoinAndClose()
	self.opts_r.Destroy()
	self.opts_r_itr.Destroy()
	for _, cf := range self.cf_handles {
		cf.Destroy()
	}
	self.cf_handle_default.Destroy()
	self.db.Close()
}

func (self *DB) GetBlockStateReader(num types.BlockNum) state_db.Reader {
	return block_state_reader{self, num}
}

type block_state_reader struct {
	*DB
	blk_n types.BlockNum
}

func (self block_state_reader) Get(col state_db.Column, k *common.Hash, cb func([]byte)) {
	if self.blk_n == types.BlockNumberNIL {
		return
	}
	if versioned_read_pool := self.versioned_read_pools[col]; versioned_read_pool != nil {
		pool_handle := versioned_read_pool.Get()
		defer versioned_read_pool.Return(pool_handle)
		ctx := pool_handle.Get().(*VersionedReadContext)
		ctx.key_buffer.SetKey(k)
		ctx.key_buffer.SetVersion(self.blk_n)
		itr := ctx.itr
		if itr.SeekForPrev(ctx.key_buffer[:]); !itr.Valid() {
			if err := itr.Err(); err != nil {
				panic(err)
			}
			return
		}
		k_slice := itr.Key()
		if bytes.HasPrefix(k_slice.Data(), k[:]) {
			v_slice := itr.Value()
			if v := v_slice.Data(); len(v) != 0 {
				cb(v)
			}
			self.maintenance_task_executor.Submit(v_slice.Free)
		}
		self.maintenance_task_executor.Submit(k_slice.Free)
		return
	}
	v_slice, err := self.db.GetCF(self.opts_r, self.cf_handles[col], k[:])
	util.PanicIfNotNil(err)
	if v := v_slice.Data(); len(v) != 0 {
		cb(v)
	}
	self.maintenance_task_executor.Submit(v_slice.Free)
}

func (self *DB) GetLatestState() state_db.LatestState {
	return &self.latest_state
}

func (self *DB) GetDPOSConfigChanges() map[uint64][]byte {
	res := make(map[uint64][]byte)

	ro := *self.opts_r
	ro.SetFillCache(false)
	it := self.db.NewIteratorCF(&ro, self.cf_handles[col_config_changes])
	defer it.Close()
	it.SeekToFirst()
	for it = it; it.Valid(); it.Next() {
		key := big.NewInt(0).SetBytes(it.Key().Data()).Uint64()
		// make a copy of bytes
		res[key] = append([]byte(nil), it.Value().Data()...)
	}
	if err := it.Err(); err != nil {
		panic(err)
	}
	return res
}

func (self *DB) SaveDPOSConfigChange(blk uint64, cfg []byte) {
	key_bytes := big.NewInt(0).SetUint64(blk).Bytes()
	self.db.PutCF(grocksdb.NewDefaultWriteOptions(), self.cf_handles[col_config_changes], key_bytes, cfg)
}

func (self *DB) invalidate_versioned_read_pools() {
	for _, col := range versioned_read_columns {
		self.maintenance_task_executor.Submit(self.versioned_read_pools[col].Invalidate())
	}
}
