package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type StaticConfig struct {
	EvmConfig      *vm.StaticConfig `json:"evm"`
	Genesis        *core.Genesis    `json:"genesis"`
	MetricsEnabled bool             `json:"metricsEnabled"`
}

type StateDBConfig struct {
	DB        *api.GenericDbConfig `json:"db"`
	CacheSize int                  `json:"cacheSize"`
}

type Config struct {
	StaticConfig
	StateDB StateDBConfig        `json:"stateDB"`
	BlockDB *api.GenericDbConfig `json:"blockDB"`
	WriteDB *api.GenericDbConfig `json:"writeDB"`
}

func (this *Config) NewVM() (ret *TaraxaVM, cleanup func(), err error) {
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
	ret = new(TaraxaVM)
	stateDb, e1 := this.StateDB.DB.NewDB()
	localErr.CheckIn(e1)
	cleanup = util.Chain(cleanup, stateDb.Close)
	blockHashDb, e2 := this.BlockDB.NewDB()
	localErr.CheckIn(e2)
	cleanup = util.Chain(cleanup, blockHashDb.Close)
	ret.ExternalApi = block_hash_db.New(blockHashDb)
	ret.StateDB = state.NewDatabaseWithCache(stateDb, this.StateDB.CacheSize)
	ret.WriteDB = stateDb
	if this.WriteDB != nil {
		writeDB, e3 := this.WriteDB.NewDB()
		localErr.CheckIn(e3)
		cleanup = util.Chain(cleanup, writeDB.Close)
		ret.WriteDB = writeDB
	}
	ret.StaticConfig = this.StaticConfig
	if ret.EvmConfig == nil {
		ret.EvmConfig = new(vm.StaticConfig)
	}
	if ret.Genesis == nil {
		ret.Genesis = core.DefaultGenesisBlock()
	}
	return
}
