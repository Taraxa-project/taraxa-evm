package trx_engine_base

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
)

type BaseTrxEngine struct {
	BaseVMConfig
	GenesisBlock *types.Block
	EvmConfig    *vm.Config
	GetBlockHash vm.GetHashFunc
	ReadDB       state.Database
	WriteDB      state.Database
	WriteDiskDB  ethdb.Database
}

func (this *BaseTrxEngine) ApplyGenesis() error {
	_, _, err := core.SetupGenesisBlock(this.WriteDiskDB, this.Genesis)
	return err
}

func (this *BaseTrxEngine) CommitToDisk(root common.Hash) error {
	return this.ReadDB.TrieDB().Commit(root, false, this.WriteDiskDB)
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
	EVMReturnValue  []byte
	NewContractAddr common.Address
	GasUsed         uint64
	ContractErr     error
	ConsensusErr    error
}

func (this *BaseTrxEngine) ExecuteTransaction(req *TransactionRequest) *TransactionResult {
	msg := types.NewMessage(
		req.Transaction.From, req.Transaction.To, uint64(req.Transaction.Nonce),
		req.Transaction.Value.ToInt(), uint64(req.Transaction.Gas),
		req.Transaction.GasPrice.ToInt(), req.Transaction.Input, req.CheckNonce)
	evmContext := vm.Context{
		GetHash:     this.GetBlockHash,
		Origin:      msg.From(),
		Coinbase:    req.BlockHeader.Miner,
		BlockNumber: req.BlockHeader.Number,
		Time:        req.BlockHeader.Time.ToInt(),
		Difficulty:  req.BlockHeader.Difficulty.ToInt(),
		GasLimit:    uint64(req.BlockHeader.GasLimit),
		GasPrice:    msg.GasPrice(),
	}
	evm := vm.NewEVMWithInterpreter(
		evmContext, req.DB, this.Genesis.Config, this.EvmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, this.EvmConfig, req.OnEvmInstruction)
		},
	)
	ret, new_contract_addr, usedGas, vmErr, consensusErr := core.
		NewStateTransition(evm, msg, req.GasPool, req.DisableMinerReward).
		TransitionDb()
	return &TransactionResult{ret, new_contract_addr, usedGas, vmErr, consensusErr}
}
