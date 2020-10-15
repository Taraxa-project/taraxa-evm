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
	db                           *gorocksdb.DB
	cf_handles                   [state_db.COL_COUNT]*gorocksdb.ColumnFamilyHandle
	col_main_trie_value_itr_pool sync.Pool
	col_acc_trie_value_itr_pool  sync.Pool
	itr_pools_mu                 sync.RWMutex
	latest_state                 latest_state
	maintenance_task_executor    goroutines.SingleThreadExecutor
	close_mu                     sync.RWMutex
	closed                       bool
}
type Opts = struct {
	Path string
}

func (self *DB) Init(opts Opts) *DB {
	const real_col_cnt = 1 + state_db.COL_COUNT
	cfnames, cfopts := [real_col_cnt]string{"default"}, [real_col_cnt]*gorocksdb.Options{cf_opts_default}
	for i := state_db.Column(1); i < real_col_cnt; i++ {
		cfnames[i], cfopts[i] = strconv.Itoa(int(i)), cf_opts_default
	}
	db, cf_handles, err := gorocksdb.OpenDbColumnFamilies(db_opts, opts.Path, cfnames[:], cfopts[:])
	util.PanicIfNotNil(err)
	self.db = db
	copy(self.cf_handles[:], cf_handles[1:])
	self.reset_itr_pools()
	self.maintenance_task_executor.Init(512) // 4KB
	self.latest_state.Init(self)
	return self
}

func (self *DB) Close() {
	defer util.LockUnlock(&self.close_mu)()
	self.latest_state.Close()
	self.maintenance_task_executor.JoinAndClose()
	self.db.Close()
	self.closed = true
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
	var itr_pool *sync.Pool
	if col == state_db.COL_acc_trie_value {
		itr_pool = &self.col_acc_trie_value_itr_pool
	} else if col == state_db.COL_main_trie_value {
		itr_pool = &self.col_main_trie_value_itr_pool
	}
	if itr_pool != nil {
		defer util.LockUnlock(self.itr_pools_mu.RLocker())()
		itr := itr_pool.Get().(*gorocksdb.Iterator)
		defer itr_pool.Put(itr)
		var k_versioned TrieValueKey
		k_versioned.SetKey(k)
		k_versioned.SetBlockNum(self.blk_n)
		if itr.SeekForPrev(k_versioned[:]); !itr.Valid() {
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
	v_slice, err := self.db.GetCF(opts_r, self.cf_handles[col], k[:])
	util.PanicIfNotNil(err)
	if v := v_slice.Data(); len(v) != 0 {
		cb(v)
	}
	self.maintenance_task_executor.Submit(v_slice.Free)
}

func (self *DB) GetLatestState() state_db.LatestState {
	return &self.latest_state
}

func (self *DB) trie_value_itr_pool(col state_db.Column) sync.Pool {
	return sync.Pool{New: func() interface{} {
		ret := self.db.NewIteratorCF(opts_r_itr, self.cf_handles[col])
		runtime.SetFinalizer(ret, func(itr *gorocksdb.Iterator) {
			defer util.LockUnlock(self.close_mu.RLocker())()
			if !self.closed {
				itr.Close()
			}
		})
		return ret
	}}
}

func (self *DB) reset_itr_pools() {
	defer util.LockUnlock(&self.itr_pools_mu)()
	self.col_main_trie_value_itr_pool = self.trie_value_itr_pool(state_db.COL_main_trie_value)
	self.col_acc_trie_value_itr_pool = self.trie_value_itr_pool(state_db.COL_acc_trie_value)
}
