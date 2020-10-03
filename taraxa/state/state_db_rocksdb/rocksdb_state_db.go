package state_db_rocksdb

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"strconv"
	"sync"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
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
	writer_thread                goroutines.SingleThreadExecutor
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
	self.writer_thread.Init(512) // TODO good parameters
	self.maintenance_task_executor.Init(512)
	self.reset_itr_pools()
	return self
}

func (self *DB) Close() {
	defer util.LockUnlock(&self.close_mu)()
	defer self.db.Close()
	self.maintenance_task_executor.Join()
	self.maintenance_task_executor.Close()
	self.writer_thread.Join()
	self.writer_thread.Close()
	self.closed = true
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

func (self *DB) ReadBlock(num types.BlockNum) state_db.ReadTx {
	return &tx_r{self, num}
}

func (self *DB) WriteBlock(num types.BlockNum) state_db.WriteTx {
	assert.Holds(num != types.BlockNumberNIL)
	var trie_key_buf TrieValueKey
	trie_key_buf.SetBlockNum(num)
	return &tx_w{tx_r{self, num - 1}, trie_key_buf, gorocksdb.NewWriteBatch()}
}

type tx_r struct {
	*DB
	blk_n types.BlockNum
}

func (self *tx_r) Get(col state_db.Column, k *common.Hash, cb func([]byte)) {
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

func (self *tx_r) NotifyDoneReading() {}

type tx_w struct {
	tx_r
	trie_value_key_buf TrieValueKey
	batch              *gorocksdb.WriteBatch
}

func (self *tx_w) Put(col state_db.Column, k *common.Hash, v []byte) {
	self.writer_thread.Submit(func() {
		if col == state_db.COL_acc_trie_value || col == state_db.COL_main_trie_value {
			self.trie_value_key_buf.SetKey(k)
			self.batch.PutCF(self.cf_handles[col], self.trie_value_key_buf[:], v)
		} else {
			self.batch.PutCF(self.cf_handles[col], k[:], v)
		}
	})
}

func (self *tx_w) Commit() (err error) {
	self.writer_thread.Submit(func() {
		err = self.db.Write(opts_w, self.batch)
		self.maintenance_task_executor.Submit(self.batch.Destroy)
		self.reset_itr_pools()
	})
	self.writer_thread.Join() // TODO completely async
	return
}

type TrieValueKey [common.HashLength + unsafe.Sizeof(types.BlockNum(0))]byte

func (self *TrieValueKey) SetKey(prefix *common.Hash) {
	copy(self[:], prefix[:])
}

func (self *TrieValueKey) SetBlockNum(block_num types.BlockNum) {
	binary.BigEndian.PutUint64(self[common.HashLength:], block_num)
}

var opts_r_itr = func() *gorocksdb.ReadOptions {
	ret := gorocksdb.NewDefaultReadOptions()
	ret.SetVerifyChecksums(false)
	ret.SetPrefixSameAsStart(true)
	ret.SetFillCache(false)
	return ret
}()
var opts_r = func() *gorocksdb.ReadOptions {
	ret := gorocksdb.NewDefaultReadOptions()
	ret.SetVerifyChecksums(false)
	return ret
}()
var opts_w = gorocksdb.NewDefaultWriteOptions()
var db_opts = func() *gorocksdb.Options {
	ret := gorocksdb.NewDefaultOptions()
	ret.SetErrorIfExists(false)
	ret.SetCreateIfMissing(true)
	ret.SetCreateIfMissingColumnFamilies(true)
	ret.IncreaseParallelism(runtime.NumCPU())
	ret.SetMaxFileOpeningThreads(runtime.NumCPU())
	ret.SetMaxOpenFiles(128)
	return ret
}()
var cf_opts_default = gorocksdb.NewDefaultOptions()
