package facade

import (
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/external_api"
	"github.com/Taraxa-project/taraxa-evm/main/state_transition"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
)

var TARAXA_CHAIN_CONFIG = params.ChainConfig{
	ChainID:             big.NewInt(0),
	HomesteadBlock:      big.NewInt(0),
	EIP150Block:         big.NewInt(0),
	EIP155Block:         big.NewInt(0),
	EIP158Block:         big.NewInt(0),
	ByzantiumBlock:      big.NewInt(0),
	ConstantinopleBlock: big.NewInt(0),
	PetersburgBlock:     big.NewInt(0),
	Ethash:              new(params.EthashConfig),
}

var TARAXA_EVM_CONFIG vm.Config

func Run(request *api.Request) (ret api.Response) {
	defer util.Recover(util.CatchAnyErr(util.SetTo(&ret.Error)))
	stateDatabase := newLdbDatabase(request.StateDatabase)
	blockchainDatabase := newLdbDatabase(request.BlockchainDatabase)
	defer stateDatabase.Close()
	defer blockchainDatabase.Close()
	taraxaEvm := state_transition.TaraxaEvm{
		StateDatabase:   state.NewDatabase(stateDatabase),
		ExternalApi:     external_api.New(blockchainDatabase),
		StateTransition: request.StateTransition,
		EvmConfig:       &TARAXA_EVM_CONFIG,
		ChainConfig:     &TARAXA_CHAIN_CONFIG,
	}
	if request.ConcurrentSchedule == nil {
		ret.ConcurrentSchedule, ret.Error = taraxaEvm.GenerateSchedule()
	} else {
		ret.ConcurrentSchedule = request.ConcurrentSchedule
		ret.StateTransitionResult, ret.Error = taraxaEvm.TransitionState(request.ConcurrentSchedule)
	}
	return
}

func newLdbDatabase(config *api.LDBConfig) *ethdb.LDBDatabase {
	db, err := ethdb.NewLDBDatabase(config.File, config.Cache, config.Handles)
	util.PanicOn(err)
	return db
}
