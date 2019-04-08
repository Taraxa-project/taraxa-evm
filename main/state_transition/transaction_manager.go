package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
	"github.com/Taraxa-project/taraxa-evm/params"
)

type TransactionManager struct {
	externalApi        *api.ExternalApi
	stateTransition    *api.StateTransition
	conflicts          *conflict_tracking.ConflictDetector
	chainConfig        *params.ChainConfig
	evmConfig          *vm.Config
	persistentDatabase ethdb.Database
}

func (this *TransactionManager) Run(schedule *api.ConcurrentSchedule) {

}
