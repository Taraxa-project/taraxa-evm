package eth_trie

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"github.com/tecbot/gorocksdb"
)

type ethdb_adapter struct {
	db    *rocksdb_ext.RocksDBExt
	col   int
	batch *gorocksdb.WriteBatch
}

func (self *ethdb_adapter) Write() error {
	return nil
}

func (self *ethdb_adapter) Get(key []byte) ([]byte, error) {
	return self.db.GetCol(self.col, key)
}

func (self *ethdb_adapter) NewBatch() ethdb.Batch {
	return self
}

func (self *ethdb_adapter) Put(key []byte, value []byte) error {
	self.db.BatchPutCol(self.batch, self.col, key, value)
	return nil
}

func (self *ethdb_adapter) Close() {
	panic("implement me")
}
