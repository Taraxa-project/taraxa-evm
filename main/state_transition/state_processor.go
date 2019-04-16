package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
)

func Run(config *api.RunConfiguration, externalApi *api.ExternalApi) (result api.Result, err error) {
	defer util.Recover(util.CatchAnyErr(func(caught error) {
		err = caught
		result.Error = err
	}))
	ldbConfig := config.LDBConfig
	ldbDatabase, ldbErr := ethdb.NewLDBDatabase(ldbConfig.File, ldbConfig.Cache, ldbConfig.Handles)
	util.PanicOn(ldbErr)
	defer ldbDatabase.Close()

	taraxaEvm := TaraxaEvm{
		readDB:          state.NewDatabase(ldbDatabase),
		stateTransition: &config.StateTransition,
		externalApi:     externalApi,
		evmConfig:       new(vm.Config),
		chainConfig: &params.ChainConfig{
			ChainID:             big.NewInt(0),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			Ethash:              new(params.EthashConfig),
		},
	}

	schedule := config.ConcurrentSchedule
	if schedule == nil {
		generatedSchedule, err := taraxaEvm.generateSchedule()
		util.PanicOn(err)
		schedule = &generatedSchedule
	}
	result.ConcurrentSchedule = schedule
	stateTransitionResult, err := taraxaEvm.transitionState(schedule)
	util.PanicOn(err)
	result.StateTransitionResult = &stateTransitionResult
	return
}
