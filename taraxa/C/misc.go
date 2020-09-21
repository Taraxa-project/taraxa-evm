package main

//#include "common.h"
import "C"
import (
	"runtime"
	"runtime/debug"

	"github.com/Taraxa-project/taraxa-evm/common/hexutil"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

//export taraxa_evm_SetGCPercent
func taraxa_evm_SetGCPercent(pct C.int) {
	debug.SetGCPercent(int(pct))
}

//export taraxa_evm_GC
func taraxa_evm_GC() {
	runtime.GC()
}

//export taraxa_evm_keccak256_InitPool
func taraxa_evm_keccak256_InitPool(size C.uint64_t) {
	keccak256.InitPool(uint64(size))
}

//export taraxa_evm_MainnetGenesisBalances
func taraxa_evm_MainnetGenesisBalances(cb C.taraxa_evm_BytesCallback) {
	call_bytes_cb(hexutil.MustDecode(core.MainnetAllocData), cb)
}

func main() {

}
