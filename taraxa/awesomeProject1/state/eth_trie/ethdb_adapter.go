package eth_trie

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"github.com/tecbot/gorocksdb"
)

type ethdb_adapter struct {
	db    *rocksdb_ext.RocksDBExt
	batch *gorocksdb.WriteBatch
}

func (self *ethdb_adapter) Write() error {
	return nil
}

func (self *ethdb_adapter) Get(key []byte) ([]byte, error) {
	return self.db.GetCol(COL_state_entries, key)
}

func (self *ethdb_adapter) NewBatch() ethdb.Batch {
	return self
}

func (self *ethdb_adapter) Put(key []byte, value []byte) error {
	self.db.BatchPutCol(self.batch, COL_state_entries, key, value)
	return nil
}

func (self *ethdb_adapter) Close() {
	panic("implement me")
}