package main

////------#---cgo CFLAGS: -I ../lib_cpp/include
////------#--include "taraxa_evm/cgo_imports.h"
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

//export Call
func Call(receiverAddr, methodName, argsEncoded *C.char) *C.char {
	ret, err := env.Call(C.GoString(receiverAddr), C.GoString(methodName), C.GoString(argsEncoded))
	util.PanicIfPresent(err)
	return C.CString(ret)
}

//export SetGCPercent
func SetGCPercent(pct C.int) {
	pctInt := int(pct)
	debug.SetGCPercent(pctInt)
	fmt.Println("SetGCPercent", pctInt)
}

//export GC
func GC() {
	runtime.GC()
}

//export Free
func Free(addr *C.char) {
	err := env.Free(C.GoString(addr))
	util.PanicIfPresent(err)
}

func main() {

}
