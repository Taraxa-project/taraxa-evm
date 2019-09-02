package base_vm

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	evm "github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
	"math/big"
)

type BaseVM struct {
	BaseVMConfig
	GenesisBlock *types.Block
	EvmConfig    *evm.Config
	GetBlockHash evm.GetHashFunc
	ReadDiskDB   *ethdb_proxy.DatabaseProxy
	WriteDiskDB  *ethdb_proxy.DatabaseProxy
	ReadDB       *state_db_proxy.DatabaseProxy
	writeDB      *state_db_proxy.DatabaseProxy
}

func (this *BaseVM) ApplyGenesis() error {
	_, _, err := core.SetupGenesisBlock(this.WriteDiskDB, this.Genesis)
	return err
}

type TransactionRequest = struct {
	Transaction      *vm.Transaction
	BlockHeader      *vm.BlockHeader
	GasPool          *core.GasPool
	CheckNonce       bool
	DB               evm.StateDB
	OnEvmInstruction evm.ExecutionController
	CanTransfer      evm.CanTransferFunc
}

type TransactionResult = struct {
	EVMReturnValue []byte
	GasUsed        uint64
	ContractErr    error
	ConsensusErr   error
}

func (this *BaseVM) ExecuteTransaction(req *TransactionRequest) *TransactionResult {
	msg := types.NewMessage(
		req.Transaction.From, req.Transaction.To, uint64(req.Transaction.Nonce), req.Transaction.Amount.ToInt(), uint64(req.Transaction.GasLimit),
		new(big.Int).Set(req.Transaction.GasPrice.ToInt()), req.Transaction.Data, req.CheckNonce)
	evmContext := evm.Context{
		CanTransfer: req.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     this.GetBlockHash,
		Origin:      msg.From(),
		Coinbase:    req.BlockHeader.Coinbase,
		BlockNumber: req.BlockHeader.Number,
		Time:        req.BlockHeader.Time.ToInt(),
		Difficulty:  req.BlockHeader.Difficulty.ToInt(),
		GasLimit:    msg.Gas(),
		GasPrice:    msg.GasPrice(),
	}
	evm := evm.NewEVMWithInterpreter(
		evmContext, req.DB, this.Genesis.Config, this.EvmConfig,
		func(vm *evm.EVM) evm.Interpreter {
			return evm.NewEVMInterpreterWithExecutionController(vm, this.EvmConfig, req.OnEvmInstruction)
		},
	)
	ret, usedGas, vmErr, consensusErr := core.NewStateTransition(evm, msg, req.GasPool).TransitionDb()
	return &TransactionResult{ret, usedGas, vmErr, consensusErr}
}
