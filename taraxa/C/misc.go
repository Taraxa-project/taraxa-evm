package main

//#include "common.h"
import "C"
import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"runtime"
	"runtime/debug"
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

func main() {

}
