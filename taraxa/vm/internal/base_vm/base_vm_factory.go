package base_vm

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type BaseVMFactory struct {
	BaseVMConfig
	EvmStaticConfig        *vm.StaticConfig `json:"evm"`
	ReadDBConfig           *StateDBConfig   `json:"readDB"`
	WriteDBConfig          *StateDBConfig   `json:"writeDB"`
	BlockHashSourceFactory `json:"blockHashSource"`
}

func (this *BaseVMFactory) NewInstance() (ret *BaseVM, cleanup func(), err error) {
	cleanup = util.DoNothing
	localErr := new(util.AtomicError)
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
	ret = &BaseVM{
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
	ret.ReadDiskDB = &ethdb_proxy.DatabaseProxy{readDiskDB, new(proxy.BaseProxy)}
	ret.ReadDB = &state_db_proxy.DatabaseProxy{
		state.NewDatabaseWithCache(ret.ReadDiskDB, this.ReadDBConfig.CacheSize),
		new(proxy.BaseProxy),
		new(proxy.BaseProxy),
	}
	ret.WriteDiskDB = ret.ReadDiskDB
	ret.WriteDB = ret.ReadDB
	if this.WriteDBConfig != nil {
		writeDiskDB, e3 := this.WriteDBConfig.DBFactory.NewInstance()
		localErr.SetOrPanicIfPresent(e3)
		cleanup = util.Chain(cleanup, writeDiskDB.Close)
		ret.WriteDiskDB = &ethdb_proxy.DatabaseProxy{writeDiskDB, new(proxy.BaseProxy)}
		ret.WriteDB = &state_db_proxy.DatabaseProxy{
			state.NewDatabaseWithCache(ret.ReadDiskDB, this.WriteDBConfig.CacheSize),
			new(proxy.BaseProxy),
			new(proxy.BaseProxy),
		}
	}
	getBlockHash, err11 := this.BlockHashSourceFactory.NewInstance()
	localErr.SetOrPanicIfPresent(err11)
	ret.GetBlockHash = getBlockHash
	return
}
