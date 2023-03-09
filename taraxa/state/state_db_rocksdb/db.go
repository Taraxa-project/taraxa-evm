package state_db_rocksdb

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"runtime"
	"strconv"

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
}

const (
	col_main_trie_value_latest = iota + state_db.COL_COUNT
	col_acc_trie_value_latest
	col_config_changes
	col_COUNT
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

func (self *DB) Prune(state_root_to_keep common.Hash, state_root_to_prune []common.Hash, blk_num types.BlockNum) {
	var member struct{}
	set_node_to_keep := make(map[common.Hash]struct{})
	set_node_to_remove := make(map[common.Hash]struct{})
	blk_num_counter := blk_num

	//Select nodes which are not to be deleted
	set_node_to_keep[state_root_to_keep] = member
	state_db.GetBlockState(self, blk_num_counter).ForEachMainNodeHashByRoot(&state_root_to_keep, func(h *common.Hash) {
		set_node_to_keep[*h] = member
	})
	blk_num_counter--

	//Select nodes from older blocks to remove only if they are not in set_node_to_keep
	for _, root_to_prune := range state_root_to_prune {
		state_db.GetBlockState(self, blk_num_counter).ForEachMainNodeHashByRoot(&root_to_prune, func(h *common.Hash) {
			if _, ok := set_node_to_keep[*h]; !ok {
				set_node_to_remove[*h] = member
			}
		})
		blk_num_counter--
	}

	//Select main trie values to prune/remove
	set_value_to_prune := make(map[string]struct{})
	set_storage_root_to_keep := make(map[common.Hash]struct{})
	set_storage_root_to_prune := make(map[common.Hash]struct{})

	itr := self.db.NewIteratorCF(self.opts_r_itr, self.cf_handles[state_db.COL_main_trie_value])
	itr.SeekToFirst()
	prev_key := make([]byte, common.VersionedKeyLength)
	last_acc_to_keep := make([]byte, common.HashLength)
	for itr.Valid() {
		account := state_db.DecodeAccountFromTrie(itr.Value().Data())
		copy(prev_key, itr.Key().Data())
		itr.Next()
		keep := true
		if itr.Valid() {
			ver_blk_num := binary.BigEndian.Uint64(itr.Key().Data()[common.HashLength:common.VersionedKeyLength])
			if bytes.Compare(prev_key[0:common.HashLength], itr.Key().Data()[0:common.HashLength]) == 0 {
				//Only prune the previous value if current value for the same account is below blk_num
				if ver_blk_num < blk_num {
					keep = false
				}
			}
		}

		if keep {
			//Only save the storage root hash of smallest block number to keep
			if bytes.Compare(prev_key[0:common.HashLength], last_acc_to_keep) != 0 {
				copy(last_acc_to_keep, prev_key[0:common.HashLength])
				if account.StorageRootHash != nil {
					set_storage_root_to_keep[*account.StorageRootHash] = member
				}
			}
		} else {
			set_value_to_prune[string(prev_key)] = member
			if account.StorageRootHash != nil {
				set_storage_root_to_prune[*account.StorageRootHash] = member
			}
		}
	}

	//Select account nodes to prune
	set_account_node_to_keep := make(map[common.Hash]struct{})
	set_account_node_to_remove := make(map[common.Hash]struct{})
	for root_to_keep, _ := range set_storage_root_to_keep {
		set_account_node_to_keep[root_to_keep] = member
		state_db.GetBlockState(self, blk_num_counter).ForEachAccountNodeHashByRoot(&root_to_keep, func(h *common.Hash) {
			set_account_node_to_keep[*h] = member
		})
	}
	for root_to_prune, _ := range set_storage_root_to_prune {
		if _, ok := set_account_node_to_keep[root_to_prune]; !ok {
			state_db.GetBlockState(self, blk_num_counter).ForEachAccountNodeHashByRoot(&root_to_prune, func(h *common.Hash) {
				if _, ok := set_account_node_to_keep[*h]; !ok {
					set_account_node_to_remove[*h] = member
				}
			})
		}
	}

	//Select account storage values to prune
	set_account_storage_value_to_keep := make(map[string]struct{})
	set_account_storage_value_to_prune := make(map[string]struct{})
	itr = self.db.NewIteratorCF(self.opts_r_itr, self.cf_handles[state_db.COL_acc_trie_value])
	itr.SeekToFirst()
	for itr.Valid() {
		copy(prev_key, itr.Key().Data())
		itr.Next()
		keep := true
		if itr.Valid() {
			ver_blk_num := binary.BigEndian.Uint64(itr.Key().Data()[common.HashLength:common.VersionedKeyLength])

			if bytes.Compare(prev_key[0:common.HashLength], itr.Key().Data()[0:common.HashLength]) == 0 {
				if ver_blk_num < blk_num {
					keep = false
				}
			}
		}
		if keep {
			set_account_storage_value_to_keep[string(prev_key)] = member
		} else {
			set_account_storage_value_to_prune[string(prev_key)] = member
		}
	}

	//Remove and compact everything we can remove
	range_limit := [common.VersionedKeyLength]byte{255}
	range32 := grocksdb.Range{
		Start: make([]byte, common.HashLength),
		Limit: make([]byte, common.HashLength),
	}
	copy(range32.Limit, range_limit[0:common.HashLength])
	range40 := grocksdb.Range{
		Start: make([]byte, common.VersionedKeyLength),
		Limit: make([]byte, common.VersionedKeyLength),
	}
	copy(range40.Limit, range_limit[0:common.VersionedKeyLength])

	for node_to_remove, _ := range set_node_to_remove {
		self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_main_trie_node], node_to_remove[:])
	}
	self.db.CompactRangeCF(self.cf_handles[state_db.COL_main_trie_node], range32)

	for value_to_remove, _ := range set_value_to_prune {
		self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_main_trie_value], []byte(value_to_remove))
	}
	self.db.CompactRangeCF(self.cf_handles[state_db.COL_main_trie_value], range40)

	for node_to_remove, _ := range set_account_node_to_remove {
		self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_acc_trie_node], node_to_remove[:])
	}
	self.db.CompactRangeCF(self.cf_handles[state_db.COL_acc_trie_node], range32)

	for value_to_remove, _ := range set_account_storage_value_to_prune {
		if _, ok := set_account_storage_value_to_keep[value_to_remove]; !ok {
			self.db.DeleteCF(self.latest_state.opts_w, self.cf_handles[state_db.COL_acc_trie_value], []byte(value_to_remove))
		}
	}
	self.db.CompactRangeCF(self.cf_handles[state_db.COL_acc_trie_value], range40)
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

func (self *DB) GetBlockState(num types.BlockNum) state_db.Reader {
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
