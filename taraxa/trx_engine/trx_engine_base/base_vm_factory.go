package trx_engine_base

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

type BaseVMConfig = struct {
	Genesis *core.Genesis `json:"genesis"`
}

type StateDBConfig = struct {
	DBFactory DBFactory `json:"db"`
}

type BlockHashSourceFactory interface {
	NewInstance() (vm.GetHashFunc, error)
}

type BaseVMFactory struct {
	BaseVMConfig
	EvmStaticConfig        *vm.StaticConfig `json:"evm"`
	ReadDBConfig           *StateDBConfig   `json:"readDB"`
	WriteDBConfig          *StateDBConfig   `json:"writeDB"`
	BlockHashSourceFactory `json:"blockHashSource"`
}

func (this *BaseVMFactory) NewInstance() (ret *BaseTrxEngine, cleanup func(), err error) {
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
		BaseVMConfig: this.BaseVMConfig,
		EvmConfig:    &vm.Config{StaticConfig: evmStaticConfig},
	}
	if ret.Genesis == nil {
		ret.Genesis = core.DefaultGenesisBlock()
	}
	ret.GenesisBlock = ret.Genesis.ToBlock(nil)
	readDiskDB, e1 := this.ReadDBConfig.DBFactory.NewInstance()
	localErr.SetOrPanicIfPresent(e1)
	cleanup = util.Chain(cleanup, readDiskDB.Close)
	ret.ReadDB = state.NewDatabase(readDiskDB)
	ret.WriteDiskDB = readDiskDB
	ret.WriteDB = ret.ReadDB
	if this.WriteDBConfig != nil {
		writeDiskDB, e3 := this.WriteDBConfig.DBFactory.NewInstance()
		localErr.SetOrPanicIfPresent(e3)
		cleanup = util.Chain(cleanup, writeDiskDB.Close)
		ret.WriteDiskDB = writeDiskDB
		ret.WriteDB = state.NewDatabase(readDiskDB)
	}
	getBlockHash, err11 := this.BlockHashSourceFactory.NewInstance()
	localErr.SetOrPanicIfPresent(err11)
	ret.GetBlockHash = getBlockHash
	return
}

type SimpleBlockHashSourceFactory vm.GetHashFunc

func (this SimpleBlockHashSourceFactory) NewInstance() (vm.GetHashFunc, error) {
	return vm.GetHashFunc(this), nil
}
