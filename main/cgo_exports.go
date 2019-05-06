package main

////------#---cgo CFLAGS: -I ../lib_cpp/include
////------#--include "taraxa_evm/cgo_imports.h"
import "C"

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_evm"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/virtual_env"
)

var env = new(virtual_env.Builder).
	Func("NewVM", func(env *virtual_env.VirtualEnv, config *api.VMConfig) (vmAddr string, err error) {
		cleanup := func() {}
		defer util.Recover(util.CatchAnyErr(func(e error) {
			panic(e)
			cleanup()
		}))
		// TODO move to the config class
		taraxaEvm := new(taraxa_evm.TaraxaVM)
		stateLDB := config.StateDB.LDB.NewLdbDatabase()
		cleanup = util.Chain(cleanup, stateLDB.Close)
		blockHashLDB := config.BlockHashLDB.NewLdbDatabase()
		cleanup = util.Chain(cleanup, blockHashLDB.Close)
		taraxaEvm.ExternalApi = block_hash_db.New(blockHashLDB)
		taraxaEvm.SourceStateDB = state.NewDatabaseWithCache(stateLDB, config.StateDB.CacheSize)
		taraxaEvm.TargetStateDB = taraxaEvm.SourceStateDB
		if config.StateTransitionTargetLDB != nil {
			ldb := config.StateTransitionTargetLDB.NewLdbDatabase()
			cleanup = util.Chain(cleanup, ldb.Close)
			taraxaEvm.TargetStateDB = state.NewDatabase(ldb)
		}
		taraxaEvm.EvmConfig = config.Evm
		if taraxaEvm.EvmConfig == nil {
			taraxaEvm.EvmConfig = new(vm.StaticConfig)
		}
		taraxaEvm.Genesis = config.Genesis
		if taraxaEvm.Genesis == nil {
			taraxaEvm.Genesis = core.DefaultGenesisBlock()
		}
		vmAddr, allocErr := env.Alloc(taraxaEvm, cleanup)
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
