package taraxa_vm

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/main/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type ThreadPoolConfig struct {
	ThreadCount int `json:"threadCount"`
	QueueSize   int `json:"queueSize"`
}

type StaticConfig struct {
	EvmConfig                           *vm.StaticConfig `json:"evm"`
	Genesis                             *core.Genesis    `json:"genesis"`
	ConflictDetectorInboxPerTransaction int              `json:"conflictDetectorInboxPerTransaction"`
	DisableEthereumBlockReward          bool             `json:"disableEthereumBlockReward"`
}

type StateDBConfig struct {
	DB        *api.GenericDBConfig `json:"db"`
	CacheSize int                  `json:"cacheSize"`
}

type VmConfig struct {
	StaticConfig
	ReadDB  *StateDBConfig       `json:"readDB"`
	WriteDB *StateDBConfig       `json:"writeDB"`
	BlockDB *api.GenericDBConfig `json:"blockDB"`
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
	rec := metric_utils.NewTimeRecorder()
	readDiskDB, e1 := this.ReadDB.DB.NewDB()
	localErr.CheckIn(e1)
	fmt.Println("create state db took", rec())
	cleanup = util.Chain(cleanup, readDiskDB.Close)
	ret.ReadDiskDB = &ethdb_proxy.DatabaseProxy{readDiskDB, new(proxy.BaseProxy)}
	ret.ReadDB = &state_db_proxy.DatabaseProxy{
		state.NewDatabaseWithCache(ret.ReadDiskDB, this.ReadDB.CacheSize),
		new(proxy.BaseProxy),
		new(proxy.BaseProxy),
	}
	if this.BlockDB != nil {
		rec = metric_utils.NewTimeRecorder()
		blockHashDb, e2 := this.BlockDB.NewDB()
		fmt.Println("create block db took", rec())
		localErr.CheckIn(e2)
		cleanup = util.Chain(cleanup, blockHashDb.Close)
		ret.ExternalApi = block_hash_db.New(blockHashDb)
	} else {
		ret.ExternalApi = &block_hash_db.NotImplementedBlockHashStore{}
	}
	ret.WriteDiskDB = ret.ReadDiskDB
	ret.WriteDB = ret.ReadDB
	if this.WriteDB != nil {
		rec = metric_utils.NewTimeRecorder()
		writeDiskDB, e3 := this.WriteDB.DB.NewDB()
		fmt.Println("create write db took", rec())
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
