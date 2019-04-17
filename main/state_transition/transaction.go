package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
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
	taraxaDb      *state_db.TaraxaStateDB
	gasPool       *core.GasPool
	executionCtrl vm.ExecutionController
}

type TransactionResult struct {
	txId           api.TxId
	value          hexutil.Bytes
	gasUsed        uint64
	logs           []*types.Log
	transientState *state_db.TransientState
	contractErr    error
	consensusErr   error
	dbErr          error
}

func (this *TransactionExecution) Run(params *TransactionParams) *TransactionResult {
	params.taraxaDb.Prepare(this.txHash, this.blockHash, this.txId)
	evmConfig := *this.evmConfig
	evm := vm.NewEVMWithInterpreter(
		*this.evmContext, params.taraxaDb, this.chainConfig, evmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, params.executionCtrl)
		},
	)
	st := core.NewStateTransition(evm, this.tx, params.gasPool)
	result := new(TransactionResult)
	result.txId = this.txId
	result.value, result.gasUsed, result.contractErr, result.consensusErr = st.TransitionDb()
	result.dbErr = params.taraxaDb.Error()
	result.logs = params.taraxaDb.GetLogs(this.txHash)
	result.transientState = params.taraxaDb.TransientState.Clone()
	return result
}
