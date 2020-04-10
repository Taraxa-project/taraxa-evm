package main

//#include "ctypes.h"
//#include "c_test_main.h"
import "C"

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_rocksdb"
	"reflect"
	"runtime"
	"runtime/debug"
	"unsafe"
)

type TaraxaStateService struct {
	db rocksdb.RocksDBStateDB
}

//export taraxa_evm_state_Prove
func taraxa_evm_state_Prove(
	blk_num_c C.uint64_t,
	state_root_c *C.taraxa_evm_Hash,
	addr_c *C.taraxa_evm_Address,
	keys_c C.taraxa_evm_Hashes,
	cb C.taraxa_evm_state_ProofCallback,
) {
	blk_num := types.BlockNum(blk_num_c)
	state_root := (*common.Hash)(unsafe.Pointer(state_root_c))
	addr := (*common.Address)(unsafe.Pointer(addr_c))
	keys := *(*[]common.Hash)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&keys_c))))
	fmt.Println(blk_num, state_root, addr, keys)
	var ret state.Proof
	//ret := state.Prove(nil, blk_num, state_root, addr, keys...)
	C.taraxa_evm_state_ProofCallbackApply(cb, (*C.taraxa_evm_state_Proof)(unsafe.Pointer(&ret)))
}

//export taraxa_cgo_SetGCPercent
func taraxa_cgo_SetGCPercent(pct C.int) {
	debug.SetGCPercent(int(pct))
}

//export taraxa_cgo_GC
func taraxa_cgo_GC() {
	runtime.GC()
}

func init() {
	//os.Setenv("GODEBUG", "cgocheck=0")
	C.test_main()
}

func main() {

}
