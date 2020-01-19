package cgo

//#include "index.h"
import "C"
import (
	"runtime"
)

type batch struct {
	db        *database
	c_self    *C.taraxa_cgo_ethdb_Batch
	valueSize int
}

func newBatch(db *database) *batch {
	ret := &batch{db: db, c_self: db.c_NewBatch()}
	runtime.SetFinalizer(ret, (*batch).c_Free)
	return ret
}

func (this *batch) c_Free() {
	C.taraxa_cgo_ethdb_Batch_Free(this.c_self)
}

func (this *batch) Put(key []byte, value []byte) error {
	keySlice, valueSlice := slice(key), slice(value)
	defer free(keySlice)
	defer free(valueSlice)
	_, err := C.taraxa_cgo_ethdb_Batch_Put(this.c_self, keySlice, valueSlice)
	if err != nil {
		return err
	}
	this.valueSize += len(value)
	return nil
}

func (this *batch) Write() error {
	_, err := C.taraxa_cgo_ethdb_Batch_Write(this.c_self)
	return err
}

func (this *batch) ValueSize() int {
	return this.valueSize
}

func (this *batch) Reset() {
	this.c_Free()
	this.c_self = this.db.c_NewBatch()
	this.valueSize = 0
}
