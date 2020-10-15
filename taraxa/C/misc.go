package main

//#include "common.h"
import "C"
import (
	"runtime"
	"runtime/debug"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/core"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

//export go_set_gc_percent
func go_set_gc_percent(pct C.int) {
	debug.SetGCPercent(int(pct))
}

//export go_gc
func go_gc() {
	runtime.GC()
}

//export go_gc_async
func go_gc_async() {
	go runtime.GC()
}

//export taraxa_evm_keccak256_init_pool
func taraxa_evm_keccak256_init_pool(size C.uint64_t) {
	keccak256.InitPool(uint64(size))
}

//export taraxa_evm_mainnet_genesis_balances
func taraxa_evm_mainnet_genesis_balances() C.taraxa_evm_Bytes {
	return go_bytes_to_c(bin.BytesView(core.MainnetAllocData))
}

func main() {

}
