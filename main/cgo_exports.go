package main

////------#---cgo CFLAGS: -I ../lib_cpp/include
////------#--include "taraxa_evm/cgo_imports.h"
import "C"

import (
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_evm"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/virtual_env"
)

var env = new(virtual_env.Builder).
	Func("NewVM", func(env *virtual_env.VirtualEnv, config *taraxa_evm.Config) (vmAddr string, err error) {
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
	}).
	Build()

//export Call
func Call(receiverAddr, methodName, argsEncoded *C.char) *C.char {
	ret, err := env.Call(C.GoString(receiverAddr), C.GoString(methodName), C.GoString(argsEncoded))
	util.PanicIfPresent(err)
	return C.CString(ret)
}

//export Free
func Free(addr *C.char) {
	err := env.Free(C.GoString(addr))
	util.PanicIfPresent(err)
}

func main() {

}
