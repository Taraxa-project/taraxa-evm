package main

////------#---cgo CFLAGS: -I ../lib_cpp/include
////------#--include "taraxa_evm/cgo_imports.h"
import "C"

import (
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/api/facade"
	"github.com/Taraxa-project/taraxa-evm/main/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_evm"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/virtual_env"
	"github.com/Taraxa-project/taraxa-evm/params"
)

var env = new(virtual_env.Builder).
	Func("NewVM", func(env *virtual_env.VirtualEnv, config *api.Config) (vmAddr, blockHashDBAddr virtual_env.Address, err error) {
		cleanup := func() {}
		defer util.Recover(util.CatchAnyErr(func(e error) {
			err = e
			cleanup()
		}))
		taraxaEvm := new(taraxa_evm.TaraxaEvm)
		stateLDB := config.StateDBConfig.LevelDB.NewLdbDatabase()
		cleanup = util.Chain(cleanup, stateLDB.Close)
		blockHashLDB := config.ExternalApiConfig.BlockHashLevelDB.NewLdbDatabase()
		cleanup = util.Chain(cleanup, blockHashLDB.Close)
		blockHashDB := block_hash_db.New(blockHashLDB)
		taraxaEvm.ExternalApi = blockHashDB
		taraxaEvm.StateDatabase = state.NewDatabaseWithCache(stateLDB, config.StateDBConfig.CacheSize)
		taraxaEvm.EvmConfig = config.EvmConfig
		if taraxaEvm.EvmConfig == nil {
			taraxaEvm.EvmConfig = new(vm.StaticConfig)
		}
		taraxaEvm.ChainConfig = config.ChainConfig
		if taraxaEvm.ChainConfig == nil {
			taraxaEvm.ChainConfig = params.MainnetChainConfig
		}
		vmAddr, allocErr := env.Alloc(&facade.TaraxaVMFacade{taraxaEvm}, cleanup)
		util.PanicOn(allocErr)
		blockHashDBAddr, allocErr = env.Alloc(blockHashDB, nil)
		util.PanicOn(allocErr)
		return
	}).
	Build()

//export Call
func Call(receiverAddr, methodName, argsEncoded *C.char) *C.char {
	ret, err := env.Call(C.GoString(receiverAddr), C.GoString(methodName), C.GoString(argsEncoded))
	util.PanicOn(err)
	return C.CString(ret)
}

//export Free
func Free(addr *C.char) {
	err := env.Free(C.GoString(addr))
	util.PanicOn(err)
}

func main() {

}
