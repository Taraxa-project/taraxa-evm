package main

//#include <stdlib.h>
//#include "db/cgo/index.h"
import "C"

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/taraxa_vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/virtual_env"
	"runtime"
	"runtime/debug"
	"unsafe"
)

// TODO refactor
var env = virtual_env.VirtualEnv{Functions: virtual_env.Functions{
	"NewVM": func(env *virtual_env.VirtualEnv, config *taraxa_vm.VmConfig) (vmAddr string, err error) {
		cleanup := util.DoNothing
		defer util.CatchAnyErr(func(e error) {
			err = e
			cleanup()
		})
		vm, vmCleanup, createErr := config.NewVM()
		cleanup = util.Chain(cleanup, vmCleanup)
		util.PanicIfPresent(createErr)
		vmAddr, allocErr := env.Alloc(vm, cleanup)
		util.PanicIfPresent(allocErr)
		return
	},
}}

//export taraxa_cgo_env_Call
func taraxa_cgo_env_Call(receiverAddr, methodName, argsEncoded *C.char) *C.char {
	ret, err := env.Call(C.GoString(receiverAddr), C.GoString(methodName), C.GoString(argsEncoded))
	util.PanicIfPresent(err)
	return C.CString(ret)
}

//export taraxa_cgo_env_Free
func taraxa_cgo_env_Free(addr *C.char) {
	err := env.Free(C.GoString(addr))
	util.PanicIfPresent(err)
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
