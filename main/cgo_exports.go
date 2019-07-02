package main

//#include "cgo_imports.h"
import "C"

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_vm"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/virtual_env"
	"runtime"
	"runtime/debug"
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

//export taraxa_cgo_Call
func taraxa_cgo_Call(receiverAddr, methodName, argsEncoded *C.char) *C.char {
	ret, err := env.Call(C.GoString(receiverAddr), C.GoString(methodName), C.GoString(argsEncoded))
	util.PanicIfPresent(err)
	return C.CString(ret)
}

//export taraxa_cgo_Free
func taraxa_cgo_Free(addr *C.char) {
	err := env.Free(C.GoString(addr))
	util.PanicIfPresent(err)
}

//export taraxa_cgo_SetGCPercent
func taraxa_cgo_SetGCPercent(pct C.int) {
	pctInt := int(pct)
	debug.SetGCPercent(pctInt)
	fmt.Println("SetGCPercent", pctInt)
}

//export taraxa_cgo_GC
func taraxa_cgo_GC() {
	runtime.GC()
}

func main() {

}
