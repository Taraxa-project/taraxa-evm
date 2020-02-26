package trx_engine_base

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
)

type BaseTrxEngine struct {
	BaseEngineConfig
	EvmConfig    *vm.Config
	GetBlockHash vm.GetHashFunc
	DB           *state.Database
}

type TransactionRequest = struct {
	Transaction        *trx_engine.Transaction
	BlockHeader        *trx_engine.BlockHeader
	GasPool            *core.GasPool
	DB                 vm.StateDB
	OnEvmInstruction   vm.ExecutionController
	CheckNonce         bool
	DisableMinerReward bool
}

type TransactionResult = struct {
	EVMReturnValue []byte
	GasUsed        uint64
	ContractErr    error
	ConsensusErr   error
}

func (self *BaseTrxEngine) ExecuteTransaction(req *TransactionRequest) *TransactionResult {
	msg := types.NewMessage(
		req.Transaction.From, req.Transaction.To, uint64(req.Transaction.Nonce),
		req.Transaction.Value.ToInt(), uint64(req.Transaction.Gas),
		req.Transaction.GasPrice.ToInt(), req.Transaction.Input, req.CheckNonce)
	evmContext := vm.Context{
		GetHash:     self.GetBlockHash,
		Origin:      msg.From(),
		Coinbase:    req.BlockHeader.Miner,
		BlockNumber: req.BlockHeader.Number,
		Time:        req.BlockHeader.Time.ToInt(),
		Difficulty:  req.BlockHeader.Difficulty.ToInt(),
		GasLimit:    uint64(req.BlockHeader.GasLimit),
		GasPrice:    msg.GasPrice(),
	}
	evm := vm.NewEVMWithInterpreter(
		evmContext, req.DB, self.Genesis.Config, self.EvmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, self.EvmConfig, req.OnEvmInstruction)
		},
	)
	ret, usedGas, vmErr, consensusErr := core.
		NewStateTransition(evm, msg, req.GasPool, req.DisableMinerReward).
		TransitionDb()
	return &TransactionResult{ret, usedGas, vmErr, consensusErr}
}
