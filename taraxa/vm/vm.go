package vm

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/taraxa/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/rendezvous"
	"math/big"
	"runtime"
	"sort"
	"sync/atomic"
)

type VM struct {
	StaticConfig
	GetBlockHash vm.GetHashFunc
	ReadDiskDB   *ethdb_proxy.DatabaseProxy
	WriteDiskDB  *ethdb_proxy.DatabaseProxy
	ReadDB       *state_db_proxy.DatabaseProxy
	WriteDB      *state_db_proxy.DatabaseProxy
}

func (this *VM) GenerateSchedule(req *StateTransitionRequest) (result *ConcurrentSchedule, metrics *ScheduleGenerationMetrics, err error) {
	result = new(ConcurrentSchedule)
	metrics = new(ScheduleGenerationMetrics)
	metrics.TransactionMetrics = make([]TransactionMetrics, len(req.Block.Transactions))
	defer metrics.TotalTime.NewTimeRecorder()()
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	txCount := len(req.Block.Transactions)
	txConflictErrors := make([]util.ErrorBarrier, txCount)
	conflictDetector := conflict_detector.New((txCount + 1) * this.ConflictDetectorInboxPerTransaction)
	conflictDetector.AddConflictHanlder(func(_ *conflict_detector.Operation, authors conflict_detector.Authors) {
		authors.Each(func(_ int, value interface{}) {
			txConflictErrors[value.(TxId)].SetIfAbsent(errors.New(""))
		})
	})
	go conflictDetector.Run()
	defer conflictDetector.Halt()
	allDone := rendezvous.New(txCount)
	parallelismFactor := 1.3 // Good
	numCPU := runtime.NumCPU()
	numProcesses := int(float64(numCPU) * parallelismFactor)
	lastScheduledTxId := int64(-1)
	for i := 0; i < numProcesses; i++ {
		go func() {
			defer util.Recover(errFatal.Catch())
			stateDB, stateDBCreateErr := state.New(req.BaseStateRoot, this.ReadDB)
			errFatal.CheckIn(stateDBCreateErr)
			for {
				txId := TxId(atomic.AddInt64(&lastScheduledTxId, 1))
				if txId >= txCount {
					break
				}
				func() {
					defer allDone.CheckIn()
					errConflict := txConflictErrors[txId]
					defer util.Recover(errConflict.Catch())
					this.executeTransaction(&transactionRequest{
						txId:        txId,
						txData:      req.Block.Transactions[txId],
						blockHeader: &req.Block.BlockHeader,
						gasPool:     new(core.GasPool).AddGas(req.Block.GasLimit),
						checkNonce:  false,
						stateDB:     &OperationLoggingStateDB{stateDB, conflictDetector.NewLogger(txId)},
						onEvmInstruction: func(pc uint64) (uint64, bool) {
							errConflict.CheckIn()
							return pc, true
						},
						canTransfer: func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
							return true
						},
						metrics: &metrics.TransactionMetrics[txId],
					})
				}()
				stateDB.Reset()
			}
		}()
	}
	allDone.Await()
	defer metrics.ConflictPostProcessing.NewTimeRecorder()()
	errFatal.CheckIn()
	conflictDetector.Halt()
	conflictingTx := conflictDetector.AwaitResult().Values()
	sort.Slice(conflictingTx, func(i, j int) bool {
		return conflictingTx[i].(TxId) < conflictingTx[j].(TxId)
	})
	result.SequentialTransactions = NewTxIdSet(conflictingTx)
	return
}

func (this *VM) TransitionState(
	req *StateTransitionRequest,
	schedule *ConcurrentSchedule,
) (
	*StateTransitionResult,
	*StateTransitionMetrics,
	error,
) {
	st := &stateTransition{
		VM:                     this,
		StateTransitionRequest: req,
		ConcurrentSchedule:     schedule,
	}
	return st.run()
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
	//defer this.ReadDiskDB.RegisterDecorator("Get", metric_utils.MeasureElapsedTime(&metrics.PersistentReads))()
	//defer this.ReadDiskDB.RegisterDecorator("Has", metric_utils.MeasureElapsedTime(&metrics.PersistentReads))()
	//defer this.ReadDB.RegisterDecorator("OpenTrie", metric_utils.MeasureElapsedTime(&metrics.TrieReads))()
	//defer this.ReadDB.RegisterDecorator("OpenStorageTrie", metric_utils.MeasureElapsedTime(&metrics.TrieReads))()
	//defer this.ReadDB.RegisterDecorator("ContractCode", metric_utils.MeasureElapsedTime(&metrics.TrieReads))()
	//defer this.ReadDB.TrieProxy.RegisterDecorator("TryGet", metric_utils.MeasureElapsedTime(&metrics.TrieReads))()
	defer metrics.TotalTime.NewTimeRecorder()()
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

//TODO
//func (this *VM) NewConflictDetector() {
//	return conflict_detector.New((txCount + 1) * this.ConflictDetectorInboxPerTransaction)
//}
