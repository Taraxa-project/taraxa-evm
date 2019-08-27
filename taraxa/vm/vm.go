package vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/state_db_proxy"
	"math/big"
)

type VM struct {
	StaticConfig
	GetBlockHash vm.GetHashFunc
	ReadDiskDB   *ethdb_proxy.DatabaseProxy
	WriteDiskDB  *ethdb_proxy.DatabaseProxy
	ReadDB       *state_db_proxy.DatabaseProxy
	writeDB      *state_db_proxy.DatabaseProxy
}

func (this *VM) GenerateSchedule(req *StateTransitionRequest) (*ConcurrentSchedule, *ScheduleGenerationMetrics, error) {
	process := newScheduleGeneration(this, req)
	process.run()
	return process.result, process.metrics, process.err.Get()
}

func (this *VM) TransitionState(
	req *StateTransitionRequest, schedule *ConcurrentSchedule,
) (
	*StateTransitionResult, *StateTransitionMetrics, error,
) {
	process := newStateTransition(this, req, schedule)
	process.run()
	return process.result, process.metrics, process.err.Get()
}

func (this *VM) RunLikeEthereum(req *StateTransitionRequest) (
	ret *StateTransitionResult,
	totalTime *metric_utils.AtomicCounter,
	err error,
) {
	st := &stateTransition{
		VM:                     this,
		StateTransitionRequest: req,
	}
	return st.RunLikeEthereum()
}

func (this *VM) TestMode(req *StateTransitionRequest, params *TestModeParams) *TestModeMetrics {
	st := &stateTransition{
		VM:                     this,
		StateTransitionRequest: req,
	}
	return st.TestMode(params)
}

func (this *VM) ApplyGenesis() (*params.ChainConfig, common.Hash, error) {
	return core.SetupGenesisBlock(this.WriteDiskDB, this.Genesis)
}

func (this *VM) GenesisRoot() common.Hash {
	return this.Genesis.ToBlock(nil).Root()
}

type transactionRequest struct {
	txId             TxId
	txData           *Transaction
	blockHeader      *BlockHeader
	gasPool          *core.GasPool
	checkNonce       bool
	stateDB          StateDB
	onEvmInstruction vm.ExecutionController
	canTransfer      vm.CanTransferFunc
	metrics          *TransactionMetrics
}

func (this *VM) executeTransaction(req *transactionRequest) *TransactionResult {
	metrics := req.metrics
	defer this.ReadDiskDB.Decorate("Get", metrics.PersistentReads.Decorator())()
	defer this.ReadDiskDB.Decorate("Has", metrics.PersistentReads.Decorator())()
	defer this.ReadDB.Decorate("OpenTrie", metrics.TrieReads.Decorator())()
	defer this.ReadDB.Decorate("OpenStorageTrie", metrics.TrieReads.Decorator())()
	defer this.ReadDB.Decorate("ContractCode", metrics.TrieReads.Decorator())()
	defer this.ReadDB.TrieProxy.Decorate("TryGet", metrics.TrieReads.Decorator())()
	defer metrics.TotalTime.Recorder()()
	block, tx, stateDB := req.blockHeader, req.txData, req.stateDB
	chainConfig := this.Genesis.Config
	blockNumber := block.Number
	evmContext := vm.Context{
		CanTransfer: req.canTransfer,
		Transfer:    core.Transfer,
		GetHash:     this.GetBlockHash,
		Origin:      tx.From,
		Coinbase:    block.Coinbase,
		BlockNumber: blockNumber,
		Time:        block.Time,
		Difficulty:  block.Difficulty,
		GasLimit:    block.GasLimit,
		GasPrice:    new(big.Int).Set(tx.GasPrice),
	}
	evmConfig := &vm.Config{
		StaticConfig: this.EvmConfig,
	}
	evm := vm.NewEVMWithInterpreter(
		evmContext, stateDB, chainConfig, evmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, req.onEvmInstruction)
		},
	)
	msg := types.NewMessage(tx.From, tx.To, tx.Nonce, tx.Amount, tx.GasLimit, tx.GasPrice, tx.Data, req.checkNonce)
	st := core.NewStateTransition(evm, msg, req.gasPool)
	stateDB.BeginTransaction(tx.Hash, block.Hash, req.txId)
	ret, usedGas, vmErr, consensusErr := st.TransitionDb()
	return &TransactionResult{req.txId, ret, usedGas, vmErr, consensusErr, stateDB.GetLogs(tx.Hash)}
}
