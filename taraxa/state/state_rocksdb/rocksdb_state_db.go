package state_rocksdb

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

type RocksDBStateDB struct {
	db                           *gorocksdb.DB
	cols                         Columns
	col_main_trie_value_itr_pool sync.Pool
	col_acc_trie_value_itr_pool  sync.Pool
	itr_pools_mu                 sync.RWMutex
	maintenance_task_executor    util.SingleThreadExecutor
	batch                        *gorocksdb.WriteBatch
	batch_accessor               util.SingleThreadExecutor
}

type Columns struct {
	COL_code                   *gorocksdb.ColumnFamilyHandle
	COL_main_trie_node         *gorocksdb.ColumnFamilyHandle
	COL_main_trie_value        *gorocksdb.ColumnFamilyHandle
	COL_main_trie_value_latest *gorocksdb.ColumnFamilyHandle
	COL_acc_trie_node          *gorocksdb.ColumnFamilyHandle
	COL_acc_trie_value         *gorocksdb.ColumnFamilyHandle
	COL_acc_trie_value_latest  *gorocksdb.ColumnFamilyHandle
}

func (self *RocksDBStateDB) Init(db *gorocksdb.DB, cols Columns) *RocksDBStateDB {
	self.db = db
	self.cols = cols
	self.reset_itr_pools()
	return self
}

func (self *RocksDBStateDB) reset_itr_pools() {
	self.col_main_trie_value_itr_pool = trie_value_itr_pool(self.db, self.cols.COL_main_trie_value)
	self.col_acc_trie_value_itr_pool = trie_value_itr_pool(self.db, self.cols.COL_acc_trie_value)
}

func (self *RocksDBStateDB) Refresh() {
	defer util.LockUnlock(&self.itr_pools_mu)()
	self.reset_itr_pools()
}

func (self *RocksDBStateDB) TransactionBegin(batch *gorocksdb.WriteBatch) {
	self.batch_accessor.Do(func() {
		self.batch = batch
	})
}

func (self *RocksDBStateDB) PutCode(code_hash *common.Hash, code []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols.COL_code, code_hash[:], code)
	})
}

func (self *RocksDBStateDB) DeleteCode(code_hash *common.Hash) {
	self.batch_accessor.Do(func() {
		self.batch.DeleteCF(self.cols.COL_code, code_hash[:])
	})
}

func (self *RocksDBStateDB) PutMainTrieNode(node_hash *common.Hash, node []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols.COL_main_trie_node, node_hash[:], node)
	})
}

func (self *RocksDBStateDB) PutMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := main_trie_val_key(block_num, addr_hash)
		self.batch.PutCF(self.cols.COL_main_trie_value, bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *RocksDBStateDB) PutMainTrieValueLatest(addr_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols.COL_main_trie_value_latest, addr_hash[:], v)
	})
}

func (self *RocksDBStateDB) DeleteMainTrieValueLatest(addr_hash *common.Hash) {
	self.batch_accessor.Do(func() {
		self.batch.DeleteCF(self.cols.COL_main_trie_value_latest, addr_hash[:])
	})
}

func (self *RocksDBStateDB) PutAccountTrieNode(node_hash *common.Hash, node []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols.COL_acc_trie_node, node_hash[:], node)
	})
}

func (self *RocksDBStateDB) PutAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_key(block_num, addr, key_hash)
		self.batch.PutCF(self.cols.COL_acc_trie_value, bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *RocksDBStateDB) PutAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_latest_key(addr, key_hash)
		self.batch.PutCF(self.cols.COL_acc_trie_value_latest, bin.AnyBytes2(unsafe.Pointer(&key), len(key)), v)
	})
}

func (self *RocksDBStateDB) DeleteAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_latest_key(addr, key_hash)
		self.batch.DeleteCF(self.cols.COL_acc_trie_value_latest, bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	})
}

func (self *RocksDBStateDB) TransactionEnd() {
	self.batch_accessor.Do(func() {
		self.batch = nil
	})
	self.batch_accessor.Synchronize()
}

func (self *RocksDBStateDB) GetCode(code_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols.COL_code, code_hash[:])
	return
}

func (self *RocksDBStateDB) GetMainTrieNode(node_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols.COL_main_trie_node, node_hash[:])
	return
}

func (self *RocksDBStateDB) GetMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash) (ret []byte) {
	key := main_trie_val_key(block_num, addr_hash)
	ret = self.find_trie_value(&self.col_main_trie_value_itr_pool, bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	return
}

func (self *RocksDBStateDB) GetMainTrieValueLatest(addr_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols.COL_main_trie_value_latest, addr_hash[:])
	return
}

func (self *RocksDBStateDB) GetAccountTrieNode(node_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols.COL_acc_trie_node, node_hash[:])
	return
}

func (self *RocksDBStateDB) GetAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash) (
	ret []byte) {
	key := acc_trie_val_key(block_num, addr, key_hash)
	ret = self.find_trie_value(&self.col_acc_trie_value_itr_pool, bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	return
}

func (self *RocksDBStateDB) GetAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash) (ret []byte) {
	key := acc_trie_val_latest_key(addr, key_hash)
	ret = self.get(self.cols.COL_acc_trie_value_latest, bin.AnyBytes2(unsafe.Pointer(&key), len(key)))
	return
}

func (self *RocksDBStateDB) get(col *gorocksdb.ColumnFamilyHandle, k []byte) (ret []byte) {
	handle, err := self.db.GetCFPinned(opts_r_default, col, k)
	util.PanicIfNotNil(err)
	ret = common.CopyBytes(handle.Data())
	self.maintenance_task_executor.Do(handle.Destroy)
	return
}

func (self *RocksDBStateDB) find_trie_value(itr_pool *sync.Pool, key []byte) (ret []byte) {
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

func trie_value_itr_pool(db *gorocksdb.DB, col *gorocksdb.ColumnFamilyHandle) sync.Pool {
	return sync.Pool{New: func() interface{} {
		ret := db.NewIteratorCF(opts_r_itr, col)
		runtime.SetFinalizer(ret, func(itr *gorocksdb.Iterator) {
			itr.Close()
		})
		return ret
	}}
}

const BlockNumberLength = int(unsafe.Sizeof(types.BlockNum(0)))

var opts_r_itr = func() *gorocksdb.ReadOptions {
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
