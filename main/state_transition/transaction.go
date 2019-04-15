package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/state_db"
	"github.com/Taraxa-project/taraxa-evm/params"
)

type TransactionExecution struct {
	txId        api.TxId
	txHash      common.Hash
	blockHash   common.Hash
	tx          core.Message
	chainConfig *params.ChainConfig
	evmContext  *vm.Context
	evmConfig   *vm.Config
}

type TransactionParams struct {
	conflictAuthor conflict_detector.Author
	stateDB        *state.StateDB
	conflicts      *conflict_detector.ConflictDetector
	gasPool        *core.GasPool
	executionCtrl  vm.ExecutionController
}

type TransactionResult struct {
	value        hexutil.Bytes
	gasUsed      uint64
	logs         []*types.Log
	contractErr  error
	consensusErr error
	dbErr        error
}

func (this *TransactionExecution) Run(params *TransactionParams) *TransactionResult {
	params.stateDB.Prepare(this.txHash, this.blockHash, int(this.txId))
	conflictTrackingDB := new(state_db.TaraxaStateDB).
		Init(params.conflictAuthor, params.stateDB, params.conflicts)
	evmConfig := *this.evmConfig
	evm := vm.NewEVMWithInterpreter(
		*this.evmContext, conflictTrackingDB, this.chainConfig, evmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, params.executionCtrl)
		},
	)
	st := core.NewStateTransition(evm, this.tx, params.gasPool)
	result := new(TransactionResult)
	result.value, result.gasUsed, result.contractErr, result.consensusErr = st.TransitionDb()
	result.dbErr = params.stateDB.Error()
	result.logs = params.stateDB.GetLogs(this.txHash)
	return result
}
