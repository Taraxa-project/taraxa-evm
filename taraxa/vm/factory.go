package vm

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
	"runtime"
)

type StaticConfig struct {
	EvmConfig                           *vm.StaticConfig `json:"evm"`
	Genesis                             *core.Genesis    `json:"genesis"`
	ConflictDetectorInboxPerTransaction int              `json:"conflictDetectorInboxPerTransaction"`
	DisableEthereumBlockReward          bool             `json:"disableEthereumBlockReward"`
	NumConcurrentProcesses              int              `json:"numConcurrentProcesses"`
	ParallelismFactor                   float32          `json:"parallelismFactor"`
}

type StateDBConfig struct {
	DB        *db.GenericFactory `json:"db"`
	CacheSize int                `json:"cacheSize"`
}

type Factory struct {
	StaticConfig
	ReadDB  *StateDBConfig     `json:"readDB"`
	WriteDB *StateDBConfig     `json:"writeDB"`
	BlockDB *db.GenericFactory `json:"blockDB"`
}

func (this *Factory) NewInstance() (ret *VM, cleanup func(), err error) {
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
	ret = &VM{
		StaticConfig: this.StaticConfig,
	}
	if ret.ConflictDetectorInboxPerTransaction == 0 {
		ret.ConflictDetectorInboxPerTransaction = 5
	}
	util.Assert(ret.NumConcurrentProcesses >= 0)
	util.Assert(ret.ParallelismFactor >= 0)
	if ret.NumConcurrentProcesses == 0 {
		if ret.ParallelismFactor == 0 {
			ret.ParallelismFactor = 1.3
		}
		numCPU := runtime.NumCPU()
		ret.NumConcurrentProcesses = int(float32(numCPU) * ret.ParallelismFactor)
		if ret.NumConcurrentProcesses < 1 {
			ret.NumConcurrentProcesses = 1
		}
	}
	if ret.EvmConfig == nil {
		ret.EvmConfig = new(vm.StaticConfig)
	}
	if ret.Genesis == nil {
		ret.Genesis = core.DefaultGenesisBlock()
	}
	readDiskDB, e1 := this.ReadDB.DB.NewInstance()
	localErr.CheckIn(e1)
	cleanup = util.Chain(cleanup, readDiskDB.Close)
	ret.ReadDiskDB = &ethdb_proxy.DatabaseProxy{readDiskDB, new(proxy.BaseProxy)}
	ret.ReadDB = &state_db_proxy.DatabaseProxy{
		state.NewDatabaseWithCache(ret.ReadDiskDB, this.ReadDB.CacheSize),
		new(proxy.BaseProxy),
		new(proxy.BaseProxy),
	}
	if this.BlockDB != nil {
		blockHashDb, e2 := this.BlockDB.NewInstance()
		localErr.CheckIn(e2)
		cleanup = util.Chain(cleanup, blockHashDb.Close)
		ret.GetBlockHash = block_hash_db.New(blockHashDb).GetHeaderHashByBlockNumber
	}
	ret.WriteDiskDB = ret.ReadDiskDB
	ret.writeDB = ret.ReadDB
	if this.WriteDB != nil {
		writeDiskDB, e3 := this.WriteDB.DB.NewInstance()
		localErr.CheckIn(e3)
		cleanup = util.Chain(cleanup, writeDiskDB.Close)
		ret.WriteDiskDB = &ethdb_proxy.DatabaseProxy{writeDiskDB, new(proxy.BaseProxy)}
		ret.writeDB = &state_db_proxy.DatabaseProxy{
			state.NewDatabaseWithCache(ret.ReadDiskDB, this.WriteDB.CacheSize),
			new(proxy.BaseProxy),
			new(proxy.BaseProxy),
		}
	}
	return
}