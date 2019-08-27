package vm

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/taraxa/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/big"
	"sort"
)

type ScheduleGenerationMetrics struct {
	TransactionMetrics     []TransactionMetrics       `json:"transactionMetrics"`
	TotalTime              metric_utils.AtomicCounter `json:"totalTime"`
	ConflictPostProcessing metric_utils.AtomicCounter `json:"conflictPostProcessing"`
}

type scheduleGeneration struct {
	*VM
	*StateTransitionRequest
	result  *ConcurrentSchedule
	metrics *ScheduleGenerationMetrics
	err     util.ErrorBarrier
}

func newScheduleGeneration(vm *VM, req *StateTransitionRequest) *scheduleGeneration {
	return &scheduleGeneration{
		VM:                     vm,
		StateTransitionRequest: req,
		result:                 new(ConcurrentSchedule),
		metrics: &ScheduleGenerationMetrics{
			TransactionMetrics: make([]TransactionMetrics, len(req.Block.Transactions)),
		},
	}
}

func (this *scheduleGeneration) run() {
	defer util.Recover(this.err.Catch())
	defer this.metrics.TotalTime.Recorder()()
	txCount := len(this.Block.Transactions)
	txConflictErrors := make([]util.ErrorBarrier, txCount)
	conflictDetector := conflict_detector.New(
		txCount*this.ConflictDetectorInboxPerTransaction,
		func(_ *conflict_detector.Operation, authors conflict_detector.Authors) {
			authors.Each(func(_ int, author conflict_detector.Author) {
				txConflictErrors[author.(TxId)].SetIfAbsent(errors.New(""))
			})
		})
	go conflictDetector.Run()
	defer conflictDetector.Halt()
	concurrent.Parallelize(this.NumConcurrentProcesses, txCount, func(int) func(int) {
		defer util.Recover(this.err.Catch())
		stateDB, stateDBCreateErr := state.New(this.BaseStateRoot, this.ReadDB)
		this.err.CheckIn(stateDBCreateErr)
		return func(txId TxId) {
			errConflict := txConflictErrors[txId]
			defer util.Recover(errConflict.Catch())
			this.executeTransaction(&transactionRequest{
				txId:        txId,
				txData:      this.Block.Transactions[txId],
				blockHeader: &this.Block.BlockHeader,
				gasPool:     new(core.GasPool).AddGas(this.Block.GasLimit),
				checkNonce:  false,
				stateDB:     &OperationLoggingStateDB{stateDB, conflictDetector.NewLogger(txId)},
				onEvmInstruction: func(pc uint64) (uint64, bool) {
					errConflict.CheckIn()
					return pc, true
				},
				canTransfer: func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
					return true
				},
				metrics: &this.metrics.TransactionMetrics[txId],
			})
		}
	})
	defer this.metrics.ConflictPostProcessing.Recorder()()
	this.err.CheckIn()
	conflictDetector.Halt()
	conflictingTx := conflictDetector.AwaitResult().Values()
	sort.Slice(conflictingTx, func(i, j int) bool {
		return conflictingTx[i].(TxId) < conflictingTx[j].(TxId)
	})
	this.result = &ConcurrentSchedule{NewTxIdSet(conflictingTx)}
}
