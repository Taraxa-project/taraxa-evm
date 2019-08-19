package cgo_db

//#include "index.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"unsafe"
)

type Config struct {
	Pointer uintptr `json:"pointer"`
}

func (this *Config) NewDB() (ret ethdb.MutableTransactionalDatabase, err error) {
	return newDatabase((*C.taraxa_cgo_ethdb_Database)(unsafe.Pointer(this.Pointer))), nil
}
