package state_db_rocksdb

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/goroutines"
	"github.com/tecbot/gorocksdb"
)

type DB struct {
	db                        *gorocksdb.DB
	cf_handle_default         *gorocksdb.ColumnFamilyHandle
	cf_handles                [col_COUNT]*gorocksdb.ColumnFamilyHandle
	opts_r                    *gorocksdb.ReadOptions
	opts_r_itr                *gorocksdb.ReadOptions
	versioned_read_pools      [col_COUNT]*util.Pool
	latest_state              LatestState
	maintenance_task_executor goroutines.GoroutineGroup
	close_mu                  sync.RWMutex
	opts                      Opts
}

const (
	col_main_trie_value_latest = iota + state_db.COL_COUNT
	col_acc_trie_value_latest
	col_COUNT
)

var versioned_read_columns = []state_db.Column{state_db.COL_acc_trie_value, state_db.COL_main_trie_value}

type Opts = struct {
	Path                            string
	DisableMostRecentTrieValueViews bool
}

func (self *DB) Init(opts Opts) *DB {
	self.opts = opts
	new_db_opts := func() *gorocksdb.Options {
		ret := gorocksdb.NewDefaultOptions()
		ret.SetErrorIfExists(false)
		ret.SetCreateIfMissing(true)
		ret.SetCreateIfMissingColumnFamilies(true)
		ret.IncreaseParallelism(runtime.NumCPU())
		ret.SetMaxFileOpeningThreads(runtime.NumCPU())
		ret.SetMaxBackgroundCompactions(runtime.NumCPU())
		ret.SetMaxBackgroundFlushes(runtime.NumCPU())
		//ret.SetEnablePipelinedWrite(true)
		//ret.SetUseAdaptiveMutex(true)
		return ret
	}
	const real_col_cnt = 1 + col_COUNT
	cf_opts_default := gorocksdb.NewDefaultOptions()
	defer cf_opts_default.Destroy()
	cfnames, cfopts := [real_col_cnt]string{"default"}, [real_col_cnt]*gorocksdb.Options{cf_opts_default}
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
	db, cf_handles, err := gorocksdb.OpenDbColumnFamilies(db_opts, opts.Path, cfnames[:], cfopts[:])
	util.PanicIfNotNil(err)
	self.db = db
	self.cf_handle_default = cf_handles[0]
	copy(self.cf_handles[:], cf_handles[1:])
	self.opts_r = func() *gorocksdb.ReadOptions {
		ret := gorocksdb.NewDefaultReadOptions()
		ret.SetVerifyChecksums(false)
		return ret
	}()
	self.opts_r_itr = func() *gorocksdb.ReadOptions {
		ret := gorocksdb.NewDefaultReadOptions()
		ret.SetVerifyChecksums(false)
		ret.SetPrefixSameAsStart(true)
		ret.SetFillCache(false)
		return ret
	}()
	self.maintenance_task_executor.Init(1, 1024) // 8KB
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
		//pool_handle := versioned_read_pool.Get()
		//defer versioned_read_pool.Return(pool_handle)
		ctx := &VersionedReadContext{
			itr: self.db.NewIteratorCF(self.opts_r_itr, self.cf_handles[col]),
		}
		defer ctx.itr.Close()
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

func (self *DB) invalidate_versioned_read_pools() {
	for _, col := range versioned_read_columns {
		self.maintenance_task_executor.Submit(self.versioned_read_pools[col].Invalidate())
	}
}
