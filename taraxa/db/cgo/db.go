package cgo

//#include "index.h"
import "C"
import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"runtime"
)

type database struct {
	ptr *C.taraxa_cgo_ethdb_Database
}

func newDatabase(ptr *C.taraxa_cgo_ethdb_Database) *database {
	ret := &database{ptr}
	runtime.SetFinalizer(ret, func(db *database) {
		C.taraxa_cgo_ethdb_Database_Free(db.ptr)
	})
	return ret
}

func (this *database) Put(key []byte, value []byte) error {
	keySlice, valueSlice := slice(key), slice(value)
	defer free(keySlice)
	defer free(valueSlice)
	cRet, cErr := C.taraxa_cgo_ethdb_Database_Put(this.ptr, keySlice, valueSlice)
	defer free(cRet)
	if cErr != nil {
		return cErr
	}
	if errBytes := bytes(cRet); len(errBytes) > 0 {
		return errors.New(string(errBytes))
	}
	return nil
}

func (this *database) Delete(key []byte) error {
	keySlice := slice(key)
	defer free(keySlice)
	cRet, cErr := C.taraxa_cgo_ethdb_Database_Delete(this.ptr, keySlice)
	defer free(cRet)
	if cErr != nil {
		return cErr
	}
	if errBytes := bytes(cRet); len(errBytes) > 0 {
		return errors.New(string(errBytes))
	}
	return nil
}

func (this *database) Get(key []byte) (ret []byte, err error) {
	keySlice := slice(key)
	defer free(keySlice)
	cRet, cErr := C.taraxa_cgo_ethdb_Database_Get(this.ptr, keySlice)
	defer free(cRet.ret)
	defer free(cRet.err)
	if cErr != nil {
		err = cErr
		return
	}
	ret = bytes(cRet.ret)
	if errBytes := bytes(cRet.err); len(errBytes) > 0 {
		err = errors.New(string(errBytes))
	}
	return
}

func (this *database) Has(key []byte) (ret bool, err error) {
	keySlice := slice(key)
	defer free(keySlice)
	cRet, cErr := C.taraxa_cgo_ethdb_Database_Has(this.ptr, keySlice)
	defer free(cRet.err)
	if cErr != nil {
		err = cErr
		return
	}
	ret = bool(cRet.ret)
	if errBytes := bytes(cRet.err); len(errBytes) > 0 {
		err = errors.New(string(errBytes))
	}
	return
}

func (this *database) Close() {
	C.taraxa_cgo_ethdb_Database_Close(this.ptr)
}

func (this *database) NewBatch() ethdb.Batch {
	return newBatch(C.taraxa_cgo_ethdb_Database_NewBatch(this.ptr))
}
