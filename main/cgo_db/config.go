package cgo_db

//#include "../cgo_imports.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"unsafe"
)

type Config struct {
	Pointer uintptr `json:"pointer"`
}

func (this *Config) NewDB() (ret ethdb.Database, err error) {
	return newDatabase((*C.taraxa_cgo_ethdb_Database)(unsafe.Pointer(this.Pointer))), nil
}
