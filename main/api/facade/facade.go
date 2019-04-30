package facade

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_evm"
)

//var TARAXA_CHAIN_CONFIG = params.ChainConfig{
//	ChainID:             big.NewInt(0),
//	HomesteadBlock:      big.NewInt(0),
//	EIP150Block:         big.NewInt(0),
//	EIP155Block:         big.NewInt(0),
//	EIP158Block:         big.NewInt(0),
//	ByzantiumBlock:      big.NewInt(0),
//	ConstantinopleBlock: big.NewInt(0),
//	PetersburgBlock:     big.NewInt(0),
//	Ethash:              new(params.EthashConfig),
//}

type TaraxaVMFacade struct {
	*taraxa_evm.TaraxaEvm
}

func (this *TaraxaVMFacade) GenerateSchedule(req *api.ScheduleRequest) (*api.ConcurrentSchedule, error) {
	//ret := new(api.ScheduleResponse)
	return this.TaraxaEvm.GenerateSchedule(req.StateTransition)
}

func (this *TaraxaVMFacade) TransitionState(req *api.StateTransitionRequest) (*api.StateTransitionResult, error) {
	var commitTo ethdb.Database
	if req.TargetLevelDB != nil {
		commitTo = req.TargetLevelDB.NewLdbDatabase()
		defer commitTo.Close()
	}
	return this.TaraxaEvm.TransitionState(req.StateTransition, req.ConcurrentSchedule, commitTo)
}
