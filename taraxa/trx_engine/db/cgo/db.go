package cgo

//#include "index.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"runtime"
)

type database struct {
	c_self *C.taraxa_cgo_ethdb_Database
}

func newDatabase(ptr *C.taraxa_cgo_ethdb_Database) *database {
	ret := &database{ptr}
	runtime.SetFinalizer(ret, (*database).c_Free)
	return ret
}

func (this *database) c_Free() {
	C.taraxa_cgo_ethdb_Database_Free(this.c_self)
}

func (this *database) Get(key []byte) ([]byte, error) {
	keySlice := slice(key)
	defer free(keySlice)
	ret, err := C.taraxa_cgo_ethdb_Database_Get(this.c_self, keySlice)
	if err != nil {
		return nil, err
	}
	defer free(ret)
	return bytes(ret), nil
}

func (this *database) c_NewBatch() *C.taraxa_cgo_ethdb_Batch {
	ret, err := C.taraxa_cgo_ethdb_Database_NewBatch(this.c_self)
	util.PanicIfNotNil(err)
	return ret
}

func (this *database) NewBatch() ethdb.Batch {
	return newBatch(this)
}

func (this *database) Close() {}
