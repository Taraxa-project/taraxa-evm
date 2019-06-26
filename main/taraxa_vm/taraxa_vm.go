package taraxa_vm

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/barrier"
	"math/big"
	"runtime"
	"sort"
	"sync/atomic"
)

type TaraxaVM struct {
	StaticConfig
	ExternalApi api.ExternalApi
	ReadDiskDB  *ethdb_proxy.DatabaseProxy
	WriteDiskDB *ethdb_proxy.DatabaseProxy
	ReadDB      *state_db_proxy.DatabaseProxy
	WriteDB     *state_db_proxy.DatabaseProxy
}

func (this *TaraxaVM) GenerateSchedule(req *api.StateTransitionRequest) (result *api.ConcurrentSchedule, metrics *ScheduleGenerationMetrics, err error) {
	result = new(api.ConcurrentSchedule)
	metrics = new(ScheduleGenerationMetrics)
	metrics.TransactionMetrics = make([]TransactionMetrics, len(req.Block.Transactions))
	defer metrics.TotalTime.NewTimeRecorder()()
	var errFatal util.ErrorBarrier
	//defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	txCount := len(req.Block.Transactions)
	txConflictErrors := make([]util.ErrorBarrier, txCount)
	conflictDetector := conflict_detector.New((txCount+1)*this.ConflictDetectorInboxPerTransaction,
		func(_ *conflict_detector.ConflictDetector, _ *conflict_detector.Operation, authors conflict_detector.Authors) {
			authors.Each(func(_ int, value interface{}) {
				txConflictErrors[value.(api.TxId)].SetIfAbsent(errors.New(""))
			})
		})
	go conflictDetector.Run()
	defer conflictDetector.SignalShutdown()
	allDone := barrier.New(txCount)
	lastScheduledTxId := int32(-1)
	parallelismFactor := 1.3 // Good
	numCPU := runtime.NumCPU()
	threadCount := int(float64(numCPU) * parallelismFactor)
	for i := 0; i < threadCount; i++ {
		go func() {
			defer util.Recover(errFatal.Catch())
			stateDB, stateDBCreateErr := state.New(req.BaseStateRoot, this.ReadDB)
			errFatal.CheckIn(stateDBCreateErr)
			for {
				txId := api.TxId(atomic.AddInt32(&lastScheduledTxId, 1))
				if txId >= txCount {
					break
				}
				// TODO move to a function:
				errConflict := txConflictErrors[txId]
				defer util.Recover(errConflict.Catch())
				defer allDone.CheckIn()
				result := this.executeTransaction(&transactionRequest{
					txId,
					req.Block.Transactions[txId],
					&req.Block.BlockHeader,
					&OperationLoggingStateDB{stateDB, conflictDetector.NewLogger(txId)},
					func(pc uint64) (uint64, bool) {
						errFatal.CheckIn()
						errConflict.CheckIn()
						return pc, true
					},
					new(core.GasPool).AddGas(req.Block.GasLimit),
					false,
					&metrics.TransactionMetrics[txId],
					func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
						return true
					},
				})
				errFatal.CheckIn(result.ConsensusErr)
				stateDB.Reset()
			}
		}()
	}
	allDone.Await()
	defer metrics.ConflictPostProcessing.NewTimeRecorder()()
	errFatal.CheckIn()
	conflictingTx := conflictDetector.SignalShutdown().AwaitResult().Values()
	sort.Slice(conflictingTx, func(i, j int) bool {
		return conflictingTx[i].(api.TxId) < conflictingTx[j].(api.TxId)
	})
	result.SequentialTransactions = api.NewTxIdSet(conflictingTx)
	return
}

func (this *TaraxaVM) TransitionState(req *api.StateTransitionRequest, schedule *api.ConcurrentSchedule) (*api.StateTransitionResult, *StateTransitionMetrics, error) {
	st := &stateTransition{
		TaraxaVM:               this,
		StateTransitionRequest: req,
		ConcurrentSchedule:     schedule,
	}
	return st.run()
}

func (this *TaraxaVM) RunLikeEthereum(req *api.StateTransitionRequest) (
	ret *api.StateTransitionResult, totalTime *metric_utils.AtomicCounter, err error,
) {
	st := &stateTransition{
		TaraxaVM:               this,
		StateTransitionRequest: req,
	}
	return st.RunLikeEthereum()
}

func (this *TaraxaVM) TestMode(req *api.StateTransitionRequest, params *TestModeParams) *TestModeMetrics {
	st := &stateTransition{
		TaraxaVM:               this,
		StateTransitionRequest: req,
	}
	return st.TestMode(params)
}

type transactionRequest struct {
	txId                  api.TxId
	txData                *api.Transaction
	blockHeader           *api.BlockHeader
	stateDB               StateDB
	interpreterController vm.ExecutionController
	gasPool               *core.GasPool
	checkNonce            bool
	metrics               *TransactionMetrics
	vm.CanTransferFunc
}

func (this *TaraxaVM) executeTransaction(req *transactionRequest) *TransactionResult {
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
		CanTransfer: req.CanTransferFunc,
		Transfer:    core.Transfer,
		GetHash:     this.ExternalApi.GetHeaderHashByBlockNumber,
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
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, req.interpreterController)
		},
	)
	st := core.NewStateTransition(evm, tx.AsMessage(req.checkNonce), req.gasPool)
	stateDB.BeginTransaction(tx.Hash, block.Hash, req.txId)
	ret, usedGas, vmErr, consensusErr := st.TransitionDb()
	return &TransactionResult{req.txId, ret, usedGas, vmErr, consensusErr, stateDB.GetLogs(tx.Hash)}
}
