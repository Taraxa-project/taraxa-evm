package cgo

//#include "index.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"unsafe"
)

type Factory struct {
	Pointer uintptr `json:"pointer"`
}

func (this *Factory) NewInstance() (ethdb.Database, error) {
	return &database{c_self: (*C.taraxa_cgo_ethdb_Database)(unsafe.Pointer(this.Pointer))}, nil
}
