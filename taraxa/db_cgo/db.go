package db_cgo

//#include "index.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"unsafe"
)

type Database struct {
	c_self *C.taraxa_cgo_ethdb_Database
}

func New(ptr uintptr) *Database {
	return &Database{c_self: (*C.taraxa_cgo_ethdb_Database)(unsafe.Pointer(ptr))}
}

func (this *Database) Get(key []byte) ([]byte, error) {
	keySlice := slice(key)
	defer free(keySlice)
	ret, err := C.taraxa_cgo_ethdb_Database_Get(this.c_self, keySlice)
	if err != nil {
		return nil, err
	}
	defer free(ret)
	return bytes(ret), nil
}

func (this *Database) c_NewBatch() *C.taraxa_cgo_ethdb_Batch {
	ret, err := C.taraxa_cgo_ethdb_Database_NewBatch(this.c_self)
	util.PanicIfNotNil(err)
	return ret
}

func (this *Database) NewBatch() ethdb.Batch {
	return newBatch(this)
}

func (this *Database) Close() {}
