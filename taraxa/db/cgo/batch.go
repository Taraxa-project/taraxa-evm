package cgo

//#include "index.h"
import "C"
import (
	"errors"
	"runtime"
)

type batch struct {
	ptr       *C.taraxa_cgo_ethdb_Batch
	valueSize int
}

func newBatch(ptr *C.taraxa_cgo_ethdb_Batch) *batch {
	ret := &batch{ptr: ptr}
	runtime.SetFinalizer(ret, func(batch *batch) {
		C.taraxa_cgo_ethdb_Batch_Free(batch.ptr)
	})
	return ret
}

func (this *batch) Put(key []byte, value []byte) error {
	keySlice, valueSlice := slice(key), slice(value)
	defer free(keySlice)
	defer free(valueSlice)
	cRet, cErr := C.taraxa_cgo_ethdb_Batch_Put(this.ptr, keySlice, valueSlice)
	defer free(cRet)
	if cErr != nil {
		return cErr
	}
	if errBytes := bytes(cRet); len(errBytes) > 0 {
		return errors.New(string(errBytes))
	}
	this.valueSize += len(value)
	return nil
}

func (this *batch) Delete(key []byte) error {
	keySlice := slice(key)
	defer free(keySlice)
	cRet, cErr := C.taraxa_cgo_ethdb_Batch_Delete(this.ptr, keySlice)
	defer free(cRet)
	if cErr != nil {
		return cErr
	}
	if errBytes := bytes(cRet); len(errBytes) > 0 {
		return errors.New(string(errBytes))
	}
	this.valueSize += 1;
	return nil
}

func (this *batch) ValueSize() int {
	return this.valueSize;
}

func (this *batch) Write() error {
	cRet, cErr := C.taraxa_cgo_ethdb_Batch_Write(this.ptr)
	defer free(cRet)
	if cErr != nil {
		return cErr
	}
	if errBytes := bytes(cRet); len(errBytes) > 0 {
		return errors.New(string(errBytes))
	}
	return nil
}

func (this *batch) Reset() {
	C.taraxa_cgo_ethdb_Batch_Reset(this.ptr)
	this.valueSize = 0
}
