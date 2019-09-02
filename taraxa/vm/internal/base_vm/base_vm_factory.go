package base_vm

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type BaseVMFactory struct {
	VmIOConfig
	BaseVMConfig
	EvmStaticConfig *vm.StaticConfig `json:"evm"`
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
	readDiskDB, e1 := this.ReadDB.DB.NewInstance()
	localErr.SetOrPanicIfPresent(e1)
	cleanup = util.Chain(cleanup, readDiskDB.Close)
	ret.ReadDiskDB = &ethdb_proxy.DatabaseProxy{readDiskDB, new(proxy.BaseProxy)}
	ret.ReadDB = &state_db_proxy.DatabaseProxy{
		state.NewDatabaseWithCache(ret.ReadDiskDB, this.ReadDB.CacheSize),
		new(proxy.BaseProxy),
		new(proxy.BaseProxy),
	}
	if this.BlockDB != nil {
		blockHashDb, e2 := this.BlockDB.NewInstance()
		localErr.SetOrPanicIfPresent(e2)
		cleanup = util.Chain(cleanup, blockHashDb.Close)
		ret.GetBlockHash = block_hash_db.New(blockHashDb).GetHeaderHashByBlockNumber
	}
	ret.WriteDiskDB = ret.ReadDiskDB
	ret.writeDB = ret.ReadDB
	if this.WriteDB != nil {
		writeDiskDB, e3 := this.WriteDB.DB.NewInstance()
		localErr.SetOrPanicIfPresent(e3)
		cleanup = util.Chain(cleanup, writeDiskDB.Close)
		ret.WriteDiskDB = &ethdb_proxy.DatabaseProxy{writeDiskDB, new(proxy.BaseProxy)}
		ret.writeDB = &state_db_proxy.DatabaseProxy{
			state.NewDatabaseWithCache(ret.ReadDiskDB, this.WriteDB.CacheSize),
			new(proxy.BaseProxy),
			new(proxy.BaseProxy),
		}
	}
	ret.GenesisBlock = this.Genesis.ToBlock(nil)
	return
}
