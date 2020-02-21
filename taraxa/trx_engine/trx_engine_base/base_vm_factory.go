package trx_engine_base

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

type BaseEngineConfig = struct {
	Genesis *core.Genesis `json:"genesis"`
}

type StateDBConfig = struct {
	DBFactory DBFactory `json:"db"`
}

type BlockHashSourceFactory interface {
	NewInstance() (vm.GetHashFunc, error)
}

type BaseEngineFactory struct {
	BaseEngineConfig
	EvmStaticConfig        *vm.StaticConfig `json:"evm"`
	DBConfig               *StateDBConfig   `json:"db"`
	BlockHashSourceFactory `json:"blockHashSource"`
}

func (this *BaseEngineFactory) NewInstance() (ret *BaseTrxEngine, cleanup func(), err error) {
	cleanup = util.DoNothing
	localErr := new(concurrent.AtomicError)
	defer util.Recover(
		func(interface{}) bool {
			cleanup()
			cleanup = util.DoNothing
			return false
		},
		localErr.Catch(util.SetTo(&err)),
	)
	evmStaticConfig := this.EvmStaticConfig
	if evmStaticConfig == nil {
		evmStaticConfig = new(vm.StaticConfig)
	}
	ret = &BaseTrxEngine{
		BaseEngineConfig: this.BaseEngineConfig,
		EvmConfig:        &vm.Config{StaticConfig: evmStaticConfig},
	}
	if ret.Genesis == nil {
		ret.Genesis = core.DefaultGenesisBlock()
	}
	db, err_1 := this.DBConfig.DBFactory.NewInstance()
	localErr.SetOrPanicIfPresent(err_1)
	cleanup = util.Chain(cleanup, db.Close)
	ret.DB = state.NewDatabase(db)
	getBlockHash, err_2 := this.BlockHashSourceFactory.NewInstance()
	localErr.SetOrPanicIfPresent(err_2)
	ret.GetBlockHash = getBlockHash
	return
}

type SimpleBlockHashSourceFactory vm.GetHashFunc

func (this SimpleBlockHashSourceFactory) NewInstance() (vm.GetHashFunc, error) {
	return vm.GetHashFunc(this), nil
}

//0xd7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544
