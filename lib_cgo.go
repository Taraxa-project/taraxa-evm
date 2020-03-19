package main

//#include <stdlib.h>
import "C"

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db_cgo"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_executor"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/managed_memory"
	"math/big"
	"runtime"
	"runtime/debug"
	"unsafe"
)

// TODO move and refactor
var env = managed_memory.ManagedMemory{Functions: managed_memory.Functions{
	"NewTaraxaTrxEngine": func(env *managed_memory.ManagedMemory, db_ptr uintptr) (addr string, err error) {
		engine := &trx_executor.TransactionExecutor{
			DB: state.NewDatabase(db_cgo.New(db_ptr)),
			GetBlockHash: func(uint64) common.Hash {
				panic("block hash by number is not implemented")
			},
			Genesis: &core.Genesis{
				// TODO consume from taraxa_node
				Config: &params.ChainConfig{
					ChainID:             big.NewInt(66),
					HomesteadBlock:      common.Big0,
					EIP150Block:         common.Big0,
					EIP155Block:         common.Big0,
					EIP158Block:         common.Big0,
					ByzantiumBlock:      common.Big0,
					ConstantinopleBlock: common.Big0,
					//PetersburgBlock:     common.Big0,
				},
			},
			DisableMinerReward: true,
			DisableNonceCheck:  true,
			DisableGasFee:      true,
		}
		return env.Alloc(engine, nil)
	},
}}

//export taraxa_cgo_env_Call
func taraxa_cgo_env_Call(receiverAddr, methodName, argsEncoded *C.char) (c_ret *C.char) {
	defer util.Recover(func(issue util.Any) {
		c_ret = C.CString(fmt.Sprint(issue))
	})
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
