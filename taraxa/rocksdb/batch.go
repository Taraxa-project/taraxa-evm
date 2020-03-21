package rocksdb

import "github.com/tecbot/gorocksdb"

type batch struct {
	db    *Database
	batch *gorocksdb.WriteBatch
}

func (self *batch) Put(key, value []byte) error {
	self.batch.Put(key, value)
	return nil
}

func (self *batch) Delete(key []byte) error {
	self.batch.Delete(key)
	return nil
}

func (self *batch) Write() error {
	defer self.cleanup()
	return self.db.db.Write(self.db.writeOpts, self.batch)
}

func (self *batch) cleanup() {
	self.batch.Destroy()
	*self = batch{}
}
