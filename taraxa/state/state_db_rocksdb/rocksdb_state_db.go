package state_db_rocksdb

import (
	"bytes"
	"encoding/binary"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
	"runtime"
	"sync"
	"unsafe"
)

type DB struct {
	db                           *gorocksdb.DB
	cols                         Columns
	col_main_trie_value_itr_pool sync.Pool
	col_acc_trie_value_itr_pool  sync.Pool
	itr_pools_mu                 sync.RWMutex
	maintenance_task_executor    util.SingleThreadExecutor
	batch                        *gorocksdb.WriteBatch
	batch_accessor               util.SingleThreadExecutor
	close_mu                     sync.RWMutex
	closed                       bool
}

type Columns = [COL_COUNT]*gorocksdb.ColumnFamilyHandle
type Column = byte

const (
	COL_code Column = iota
	COL_main_trie_node
	COL_main_trie_value
	COL_main_trie_value_latest
	COL_acc_trie_node
	COL_acc_trie_value
	COL_acc_trie_value_latest
	COL_COUNT
)

func (self *DB) Init(db *gorocksdb.DB, cols Columns) *DB {
	self.db = db
	self.cols = cols
	self.reset_itr_pools()
	return self
}

func (self *DB) Close() {
	defer util.LockUnlock(&self.close_mu)()
	self.maintenance_task_executor.Synchronize()
	self.batch_accessor.Synchronize()
	self.closed = true
}

func (self *DB) reset_itr_pools() {
	self.col_main_trie_value_itr_pool = self.trie_value_itr_pool(COL_main_trie_value)
	self.col_acc_trie_value_itr_pool = self.trie_value_itr_pool(COL_acc_trie_value)
}

func (self *DB) TransactionBegin(batch *gorocksdb.WriteBatch) {
	self.batch_accessor.Do(func() {
		self.batch = batch
	})
}

func (self *DB) TransactionEnd() {
	self.batch_accessor.Do(func() {
		self.batch = nil
	})
	self.batch_accessor.Synchronize()
}

func (self *DB) Refresh() {
	defer util.LockUnlock(&self.itr_pools_mu)()
	self.reset_itr_pools()
}

func (self *DB) PutCode(code_hash *common.Hash, code []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols[COL_code], code_hash[:], code)
	})
}

func (self *DB) DeleteCode(code_hash *common.Hash) {
	self.batch_accessor.Do(func() {
		self.batch.DeleteCF(self.cols[COL_code], code_hash[:])
	})
}

func (self *DB) PutMainTrieNode(node_hash *common.Hash, node []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols[COL_main_trie_node], node_hash[:], node)
	})
}

func (self *DB) PutMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := main_trie_val_key(block_num, addr_hash)
		self.batch.PutCF(self.cols[COL_main_trie_value], bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *DB) PutMainTrieValueLatest(addr_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols[COL_main_trie_value_latest], addr_hash[:], v)
	})
}

func (self *DB) DeleteMainTrieValueLatest(addr_hash *common.Hash) {
	self.batch_accessor.Do(func() {
		self.batch.DeleteCF(self.cols[COL_main_trie_value_latest], addr_hash[:])
	})
}

func (self *DB) PutAccountTrieNode(node_hash *common.Hash, node []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols[COL_acc_trie_node], node_hash[:], node)
	})
}

func (self *DB) PutAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_key(block_num, addr, key_hash)
		self.batch.PutCF(self.cols[COL_acc_trie_value], bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *DB) PutAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_latest_key(addr, key_hash)
		self.batch.PutCF(self.cols[COL_acc_trie_value_latest], bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *DB) DeleteAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_latest_key(addr, key_hash)
		self.batch.DeleteCF(self.cols[COL_acc_trie_value_latest], bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	})
}

func (self *DB) GetCode(code_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols[COL_code], code_hash[:])
	return
}

func (self *DB) GetMainTrieNode(node_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols[COL_main_trie_node], node_hash[:])
	return
}

func (self *DB) GetMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash) (ret []byte) {
	key := main_trie_val_key(block_num, addr_hash)
	ret = self.find_trie_value(&self.col_main_trie_value_itr_pool, bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	return
}

func (self *DB) GetMainTrieValueLatest(addr_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols[COL_main_trie_value_latest], addr_hash[:])
	return
}

func (self *DB) GetAccountTrieNode(node_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols[COL_acc_trie_node], node_hash[:])
	return
}

func (self *DB) GetAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash) (
	ret []byte) {
	key := acc_trie_val_key(block_num, addr, key_hash)
	ret = self.find_trie_value(&self.col_acc_trie_value_itr_pool, bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	return
}

func (self *DB) GetAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash) (ret []byte) {
	key := acc_trie_val_latest_key(addr, key_hash)
	ret = self.get(self.cols[COL_acc_trie_value_latest], bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	return
}

func (self *DB) get(col *gorocksdb.ColumnFamilyHandle, k []byte) (ret []byte) {
	handle, err := self.db.GetCFPinned(opts_r_default, col, k)
	util.PanicIfNotNil(err)
	ret = common.CopyBytes(handle.Data())
	self.maintenance_task_executor.Do(handle.Destroy)
	return
}

func (self *DB) find_trie_value(itr_pool *sync.Pool, key []byte) (ret []byte) {
	defer util.LockUnlock(self.itr_pools_mu.RLocker())()
	itr := itr_pool.Get().(*gorocksdb.Iterator)
	defer itr_pool.Put(itr)
	if itr.SeekForPrev(key); !itr.Valid() {
		if err := itr.Err(); err != nil {
			panic(err)
		}
		return
	}
	k_slice := itr.Key()
	if bytes.HasPrefix(k_slice.Data(), key[:len(key)-BlockNumberLength]) {
		v_slice := itr.Value()
		ret = common.CopyBytes(v_slice.Data())
		self.maintenance_task_executor.Do(v_slice.Free)
	}
	self.maintenance_task_executor.Do(k_slice.Free)
	return
}

func main_trie_val_key(block_num types.BlockNum, addr_hash *common.Hash) (
	ret [common.HashLength + BlockNumberLength]byte) {
	copy(ret[:], addr_hash[:])
	binary.BigEndian.PutUint64(ret[common.HashLength:], block_num)
	return
}

func acc_trie_val_key(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash) (
	ret [common.AddressLength + common.HashLength + BlockNumberLength]byte) {
	copy(ret[:], addr[:])
	copy(ret[common.AddressLength:], key_hash[:])
	binary.BigEndian.PutUint64(ret[common.AddressLength+common.HashLength:], block_num)
	return
}

func acc_trie_val_latest_key(addr *common.Address, key_hash *common.Hash) (
	ret [common.AddressLength + common.HashLength]byte) {
	copy(ret[:], addr[:])
	copy(ret[common.AddressLength:], key_hash[:])
	return
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
