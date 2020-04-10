package state_rocksdb

import (
	"bytes"
	"encoding/binary"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/tecbot/gorocksdb"
	"runtime"
	"sync"
	"unsafe"
)

type RocksDBStateDB struct {
	db                           *gorocksdb.DB
	cols                         Columns
	maintenance_task_executor    util.SingleThreadExecutor
	col_main_trie_value_itr_pool sync.Pool
	col_acc_trie_value_itr_pool  sync.Pool
	batch                        *gorocksdb.WriteBatch
	batch_accessor               util.SingleThreadExecutor
	util.InitFlag
}
type Columns struct {
	COL_code            *gorocksdb.ColumnFamilyHandle
	COL_main_trie_node  *gorocksdb.ColumnFamilyHandle
	COL_main_trie_value *gorocksdb.ColumnFamilyHandle
	COL_acc_trie_node   *gorocksdb.ColumnFamilyHandle
	COL_acc_trie_value  *gorocksdb.ColumnFamilyHandle
}

func (self *RocksDBStateDB) I(db *gorocksdb.DB, cols Columns) *RocksDBStateDB {
	self.InitOnce()
	self.db = db
	self.cols = cols
	self.col_main_trie_value_itr_pool = trie_value_itr_pool(self.db, cols.COL_main_trie_value)
	self.col_acc_trie_value_itr_pool = trie_value_itr_pool(self.db, cols.COL_acc_trie_value)
	return self
}

func (self *RocksDBStateDB) BatchBegin(batch *gorocksdb.WriteBatch) {
	self.batch = batch
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
		self.batch.PutCF(self.cols.COL_main_trie_value, key[:], v)
	})
}

func (self *RocksDBStateDB) PutAccountTrieNode(node_hash *common.Hash, node []byte) {
	self.batch_accessor.Do(func() {
		self.batch.PutCF(self.cols.COL_acc_trie_node, node_hash[:], node)
	})
}

func (self *RocksDBStateDB) PutAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash, v []byte) {
	self.batch_accessor.Do(func() {
		key := acc_trie_val_key(block_num, key_hash, addr)
		self.batch.PutCF(self.cols.COL_acc_trie_value, key[:], v)
	})
}

func (self *RocksDBStateDB) BatchDone() {
	self.batch_accessor.Join()
	self.batch = nil
}

func (self *RocksDBStateDB) GetCode(code_hash *common.Hash) []byte {
	return self.get(self.cols.COL_code, code_hash[:])
}

func (self *RocksDBStateDB) GetMainTrieNode(node_hash *common.Hash) []byte {
	return self.get(self.cols.COL_main_trie_node, node_hash[:])
}

func (self *RocksDBStateDB) GetMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash) (ret []byte) {
	k := main_trie_val_key(block_num, addr_hash)
	v_slice := self.find_trie_value(&self.col_main_trie_value_itr_pool, k[:])
	if v_slice == nil {
		return
	}
	ret = common.CopyBytes(v_slice.Data())
	self.maintenance_task_executor.Do(v_slice.Free)
	return
}

func (self *RocksDBStateDB) GetAccountTrieNode(node_hash *common.Hash) (ret []byte) {
	ret = self.get(self.cols.COL_acc_trie_node, node_hash[:])
	return
}

func (self *RocksDBStateDB) GetAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash) (ret []byte) {
	key := acc_trie_val_key(block_num, key_hash, addr)
	v_slice := self.find_trie_value(&self.col_acc_trie_value_itr_pool, key[:])
	if v_slice == nil {
		return
	}
	ret = common.CopyBytes(v_slice.Data())
	self.maintenance_task_executor.Do(v_slice.Free)
	return
}

func (self *RocksDBStateDB) get(col *gorocksdb.ColumnFamilyHandle, k []byte) (ret []byte) {
	handle, err := self.db.GetCFPinned(opts_r_default, col, k)
	util.PanicIfNotNil(err)
	ret = common.CopyBytes(handle.Data())
	self.maintenance_task_executor.Do(handle.Destroy)
	return
}

func (self *RocksDBStateDB) find_trie_value(itr_pool *sync.Pool, key []byte) (v_slice *gorocksdb.Slice) {
	itr := itr_pool.Get().(*gorocksdb.Iterator)
	//defer itr_pool.Put(itr)
	if itr.SeekForPrev(key); !itr.Valid() {
		if err := itr.Err(); err != nil {
			panic(err)
		}
		return
	}
	k_slice := itr.Key()
	if bytes.HasPrefix(k_slice.Data(), key[:len(key)-BlockNumberLength]) {
		v_slice = itr.Value()
	}
	self.maintenance_task_executor.Do(k_slice.Free)
	return
}

func main_trie_val_key(block_num types.BlockNum, addr_hash *common.Hash) (ret main_trie_key) {
	copy(ret[:], addr_hash[:])
	binary.BigEndian.PutUint64(ret[common.HashLength:], block_num)
	return ret
}

func acc_trie_val_key(block_num types.BlockNum, key_hash *common.Hash, addr *common.Address) (ret acc_trie_key) {
	copy(ret[:], addr[:])
	copy(ret[common.AddressLength:], key_hash[:])
	binary.BigEndian.PutUint64(ret[common.AddressLength+common.HashLength:], block_num)
	return ret
}

type main_trie_key = [common.HashLength + BlockNumberLength]byte
type acc_trie_key = [common.AddressLength + common.HashLength + BlockNumberLength]byte

const BlockNumberLength = int(unsafe.Sizeof(types.BlockNum(0)))

func trie_value_itr_pool(db *gorocksdb.DB, col *gorocksdb.ColumnFamilyHandle) sync.Pool {
	return sync.Pool{New: func() interface{} {
		opts := gorocksdb.NewDefaultReadOptions()
		opts.SetPrefixSameAsStart(true)
		ret := db.NewIteratorCF(opts, col)
		runtime.SetFinalizer(ret, func(self *gorocksdb.Iterator) {
			self.Close()
			opts.Destroy()
		})
		return ret
	}}
}

var opts_r_default = gorocksdb.NewDefaultReadOptions()
