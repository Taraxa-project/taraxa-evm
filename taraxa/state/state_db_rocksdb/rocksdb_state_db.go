package state_db_rocksdb

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"sync"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/goroutines"
	"github.com/tecbot/gorocksdb"
)

type DB struct {
	db                           *gorocksdb.DB
	cols                         Columns
	col_main_trie_value_itr_pool sync.Pool
	col_acc_trie_value_itr_pool  sync.Pool
	itr_pools_mu                 sync.RWMutex
	maintenance_task_executor    goroutines.SingleThreadExecutor
	batch                        *gorocksdb.WriteBatch
	batch_accessor               goroutines.SingleThreadExecutor
	close_mu                     sync.RWMutex
	closed                       bool
}

type Columns = [COL_COUNT]*gorocksdb.ColumnFamilyHandle
type Column = byte

const (
	COL_code Column = iota
	COL_main_trie_node
	COL_main_trie_value
	COL_acc_trie_node
	COL_acc_trie_value
	COL_COUNT
)

func (self *DB) Init(db *gorocksdb.DB, cols Columns) *DB {
	self.db = db
	self.cols = cols
	self.batch_accessor.Init(512)
	self.maintenance_task_executor.Init(512)
	self.reset_itr_pools()
	return self
}

func (self *DB) Close() {
	defer util.LockUnlock(&self.close_mu)()
	self.maintenance_task_executor.Join()
	self.maintenance_task_executor.Close()
	self.batch_accessor.Join()
	self.batch_accessor.Close()
	self.closed = true
}

func (self *DB) trie_value_itr_pool(col Column) sync.Pool {
	return sync.Pool{New: func() interface{} {
		ret := self.db.NewIteratorCF(opts_r_default_itr, self.cols[col])
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
	self.col_main_trie_value_itr_pool = self.trie_value_itr_pool(COL_main_trie_value)
	self.col_acc_trie_value_itr_pool = self.trie_value_itr_pool(COL_acc_trie_value)
}

func (self *DB) BatchBegin(batch *gorocksdb.WriteBatch) {
	self.batch_accessor.Submit(func() {
		self.batch = batch
	})
}

func (self *DB) BatchEnd() {
	self.batch_accessor.Submit(func() {
		self.batch = nil
	})
	self.batch_accessor.Join()
}

func (self *DB) Refresh() {
	defer util.LockUnlock(&self.itr_pools_mu)()
	self.reset_itr_pools()
}

func (self *DB) get(col *gorocksdb.ColumnFamilyHandle, k []byte, cb func([]byte)) {
	handle, err := self.db.GetCFPinned(opts_r_default, col, k)
	util.PanicIfNotNil(err)
	if v := handle.Data(); len(v) != 0 {
		cb(v)
	}
	self.maintenance_task_executor.Submit(handle.Destroy)
	return
}

func (self *DB) NewBlockReadTransaction(num types.BlockNum) state_common.BlockReadTransaction {
	return &block_view{self, num}
}

func (self *DB) NewBlockCreationTransaction(num types.BlockNum) state_common.BlockCreationTransaction {
	return &block_view{self, num - 1}
}

type block_view struct {
	*DB
	read_block_num types.BlockNum
}

func (self *block_view) PutCode(k *common.Hash, v []byte) {
	self.batch_accessor.Submit(func() {
		self.batch.PutCF(self.cols[COL_code], k[:], v)
	})
}

func (self *block_view) PutMainTrieNode(k *common.Hash, v []byte) {
	self.batch_accessor.Submit(func() {
		self.batch.PutCF(self.cols[COL_main_trie_node], k[:], v)
	})
}

func (self *block_view) PutMainTrieValue(k *common.Hash, v []byte) {
	self.put_trie_value(COL_main_trie_value, k, v)
}

func (self *block_view) PutAccountTrieNode(k *common.Hash, v []byte) {
	self.batch_accessor.Submit(func() {
		self.batch.PutCF(self.cols[COL_acc_trie_node], k[:], v)
	})
}

func (self *block_view) PutAccountTrieValue(k *common.Hash, v []byte) {
	self.put_trie_value(COL_acc_trie_value, k, v)
}

func (self *block_view) GetCode(k *common.Hash) state_common.ManagedSlice {
	handle, err := self.db.GetCFPinned(opts_r_default, self.cols[COL_code], k[:])
	util.PanicIfNotNil(err)
	return new(managed_slice).Init(handle)
}

func (self *block_view) GetMainTrieNode(k *common.Hash, cb func([]byte)) {
	self.get(self.cols[COL_main_trie_node], k[:], cb)
}

func (self *block_view) GetMainTrieValue(k *common.Hash, cb func([]byte)) {
	self.find_trie_value(&self.col_main_trie_value_itr_pool, k, cb)
}

func (self *block_view) GetAccountTrieNode(k *common.Hash, cb func([]byte)) {
	self.get(self.cols[COL_acc_trie_node], k[:], cb)
}

func (self *block_view) GetAccountTrieValue(k *common.Hash, cb func([]byte)) {
	self.find_trie_value(&self.col_acc_trie_value_itr_pool, k, cb)
}

func (self *block_view) NotifyDoneReading() {}

func (self *block_view) put_trie_value(col Column, k *common.Hash, v []byte) {
	self.batch_accessor.Submit(func() {
		key := versioned_key(k, self.read_block_num+1)
		self.batch.PutCF(self.cols[col], bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *block_view) find_trie_value(itr_pool *sync.Pool, k *common.Hash, cb func([]byte)) {
	defer util.LockUnlock(self.itr_pools_mu.RLocker())()
	itr := itr_pool.Get().(*gorocksdb.Iterator)
	defer itr_pool.Put(itr)
	k_versioned := versioned_key(k, self.read_block_num)
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
}

func versioned_key(k *common.Hash, block_num types.BlockNum) (ret [common.HashLength + BlockNumberLength]byte) {
	copy(ret[:], k[:])
	binary.BigEndian.PutUint64(ret[common.HashLength:], block_num)
	return
}

const BlockNumberLength = int(unsafe.Sizeof(types.BlockNum(0)))

var opts_r_default_itr = func() *gorocksdb.ReadOptions {
	ret := gorocksdb.NewDefaultReadOptions()
	ret.SetVerifyChecksums(false)
	ret.SetPrefixSameAsStart(true)
	ret.SetFillCache(false)
	return ret
}()

var opts_r_default = func() *gorocksdb.ReadOptions {
	ret := gorocksdb.NewDefaultReadOptions()
	ret.SetVerifyChecksums(false)
	return ret
}()
