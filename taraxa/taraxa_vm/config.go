package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type StaticConfig struct {
	EvmConfig                           *vm.StaticConfig `json:"evm"`
	Genesis                             *core.Genesis    `json:"genesis"`
	ConflictDetectorInboxPerTransaction int              `json:"conflictDetectorInboxPerTransaction"`
	DisableEthereumBlockReward          bool             `json:"disableEthereumBlockReward"`
}

type StateDBConfig struct {
	DB        *db.GenericFactory `json:"db"`
	CacheSize int                `json:"cacheSize"`
}

type VmConfig struct {
	StaticConfig
	ReadDB  *StateDBConfig     `json:"readDB"`
	WriteDB *StateDBConfig     `json:"writeDB"`
	BlockDB *db.GenericFactory `json:"blockDB"`
}

func (this *VmConfig) NewVM() (ret *TaraxaVM, cleanup func(), err error) {
	cleanup = util.DoNothing
	localErr := new(util.ErrorBarrier)
	defer util.Recover(
		func(interface{}) bool {
			cleanup()
			cleanup = util.DoNothing
			return false
		},
		localErr.Catch(util.SetTo(&err)),
	)
	ret = &TaraxaVM{
		StaticConfig: this.StaticConfig,
	}
	ret.ConflictDetectorInboxPerTransaction = util.Max(ret.ConflictDetectorInboxPerTransaction, 100)
	if ret.EvmConfig == nil {
		ret.EvmConfig = new(vm.StaticConfig)
	}
	if ret.Genesis == nil {
		ret.Genesis = core.DefaultGenesisBlock()
	}
	readDiskDB, e1 := this.ReadDB.DB.NewDB()
	localErr.CheckIn(e1)
	cleanup = util.Chain(cleanup, readDiskDB.Close)
	ret.ReadDiskDB = &ethdb_proxy.DatabaseProxy{readDiskDB, new(proxy.BaseProxy)}
	ret.ReadDB = &state_db_proxy.DatabaseProxy{
		state.NewDatabaseWithCache(ret.ReadDiskDB, this.ReadDB.CacheSize),
		new(proxy.BaseProxy),
		new(proxy.BaseProxy),
	}
	if this.BlockDB != nil {
		blockHashDb, e2 := this.BlockDB.NewDB()
		localErr.CheckIn(e2)
		cleanup = util.Chain(cleanup, blockHashDb.Close)
		ret.BlockHashStore = block_hash_db.New(blockHashDb)
	} else {
		ret.BlockHashStore = &block_hash_db.NotImplementedBlockHashStore{}
	}
	ret.WriteDiskDB = ret.ReadDiskDB
	ret.WriteDB = ret.ReadDB
	if this.WriteDB != nil {
		writeDiskDB, e3 := this.WriteDB.DB.NewDB()
		localErr.CheckIn(e3)
		cleanup = util.Chain(cleanup, writeDiskDB.Close)
		ret.WriteDiskDB = &ethdb_proxy.DatabaseProxy{writeDiskDB, new(proxy.BaseProxy)}
		ret.WriteDB = &state_db_proxy.DatabaseProxy{
			state.NewDatabaseWithCache(ret.ReadDiskDB, this.WriteDB.CacheSize),
			new(proxy.BaseProxy),
			new(proxy.BaseProxy),
		}
	}
	return
}
