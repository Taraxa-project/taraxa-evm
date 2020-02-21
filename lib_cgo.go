package main

//#include <stdlib.h>
import "C"

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/db/cgo"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_eth"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/managed_memory"
	"runtime"
	"runtime/debug"
	"unsafe"
)

var env = managed_memory.ManagedMemory{Functions: managed_memory.Functions{
	"NewTaraxaTrxEngine": func(env *managed_memory.ManagedMemory, db_ptr uintptr) (addr string, err error) {
		var factory trx_engine_eth.EthTrxEngineFactory
		factory.Genesis = trx_engine.TaraxaGenesisConfig
		factory.DisableMinerReward = true
		factory.DisableNonceCheck = true
		factory.DisableGasFee = true
		factory.DBConfig = &trx_engine_base.StateDBConfig{
			DBFactory: &cgo.Factory{Pointer: db_ptr},
		}
		factory.BlockHashSourceFactory = trx_engine_base.SimpleBlockHashSourceFactory(func(uint64) (ret common.Hash) {
			panic("block hash by number is not implemented")
		})
		val, cleanup, initErr := factory.NewInstance()
		if err = initErr; err != nil {
			return
		}
		// TODO maybe always use concurrent access?
		addr, err = env.Alloc(val, cleanup)
		return
	},
}}

//export taraxa_cgo_env_Call
func taraxa_cgo_env_Call(receiverAddr, methodName, argsEncoded *C.char) *C.char {
	ret, err := env.Call(C.GoString(receiverAddr), C.GoString(methodName), C.GoString(argsEncoded))
	util.PanicIfNotNil(err)
	return C.CString(ret)
}

//export taraxa_cgo_env_Free
func taraxa_cgo_env_Free(addr *C.char) {
	err := env.Free(C.GoString(addr))
	util.PanicIfNotNil(err)
}

//export taraxa_cgo_SetGCPercent
func taraxa_cgo_SetGCPercent(pct C.int) {
	debug.SetGCPercent(int(pct))
}

//export taraxa_cgo_GC
func taraxa_cgo_GC() {
	runtime.GC()
}

//export taraxa_cgo_malloc
func taraxa_cgo_malloc(size C.size_t) unsafe.Pointer {
	return C.malloc(size)
}

//export taraxa_cgo_free
func taraxa_cgo_free(pointer unsafe.Pointer) {
	C.free(pointer)
}

func main() {

}
