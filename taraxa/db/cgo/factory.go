package cgo

//#include "types.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"unsafe"
)

type Factory struct {
	Pointer uintptr `json:"pointer"`
}

func (this *Factory) NewInstance() (ret ethdb.MutableTransactionalDatabase, err error) {
	return newDatabase((*C.taraxa_cgo_ethdb_Database)(unsafe.Pointer(this.Pointer))), nil
}
