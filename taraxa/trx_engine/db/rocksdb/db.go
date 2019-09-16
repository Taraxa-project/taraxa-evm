package rocksdb

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/tecbot/gorocksdb"
)

type Database struct {
	writeOpts *gorocksdb.WriteOptions
	readOpts  *gorocksdb.ReadOptions
	db        *gorocksdb.DB
}

func (this *Database) GetDB() *gorocksdb.DB {
	return this.db
}

func (this *Database) Put(key []byte, value []byte) error {
	return this.db.Put(this.writeOpts, key, value)
}

func (this *Database) Delete(key []byte) error {
	return this.db.Delete(this.writeOpts, key)
}

func (this *Database) Get(key []byte) ([]byte, error) {
	return this.db.GetBytes(this.readOpts, key)
}

func (this *Database) Close() {
	this.db.Close()
	this.db = nil
}

func (this *Database) NewBatch() ethdb.Batch {
	return &batch{
		db:    this,
		batch: gorocksdb.NewWriteBatch(),
	}
}
