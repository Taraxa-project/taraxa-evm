package rocksdb

import "github.com/tecbot/gorocksdb"

type batch struct {
	db    *Database
	batch *gorocksdb.WriteBatch
	size  int
}

func (this *batch) Put(key, value []byte) error {
	this.batch.Put(key, value)
	this.size += len(value)
	return nil
}

func (this *batch) Delete(key []byte) error {
	this.batch.Delete(key)
	this.size += 1
	return nil
}

func (this *batch) Write() error {
	defer this.cleanup()
	return this.db.db.Write(this.db.writeOpts, this.batch)
}

func (this *batch) ValueSize() int {
	return this.size
}

func (this *batch) Reset() {
	this.cleanup()
	this.batch = gorocksdb.NewWriteBatch()
}

func (this *batch) cleanup() {
	if this.batch != nil {
		this.batch.Destroy()
		this.batch = nil
	}
	this.size = 0
}
