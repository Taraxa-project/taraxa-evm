package taraxa_vm

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
	vm2 "github.com/Taraxa-project/taraxa-evm/taraxa/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"
	"math/big"
	"sort"
)

type ScheduleGenerationMetrics struct {
	TransactionMetrics     []vm2.TransactionMetrics   `json:"transactionMetrics"`
	TotalTime              metric_utils.AtomicCounter `json:"totalTime"`
	ConflictPostProcessing metric_utils.AtomicCounter `json:"conflictPostProcessing"`
}

type scheduleGeneration struct {
	*TaraxaVM
	*vm2.StateTransitionRequest
	result  *vm2.ConcurrentSchedule
	metrics *ScheduleGenerationMetrics
	err     util.AtomicError
}

func newScheduleGeneration(vm *TaraxaVM, req *vm2.StateTransitionRequest) *scheduleGeneration {
	return &scheduleGeneration{
		TaraxaVM:               vm,
		StateTransitionRequest: req,
		result:                 new(vm2.ConcurrentSchedule),
		metrics: &ScheduleGenerationMetrics{
			TransactionMetrics: make([]vm2.TransactionMetrics, len(req.Block.Transactions)),
		},
	}
}

func (this *scheduleGeneration) run() {
	defer util.Recover(this.err.Catch())
	defer this.metrics.TotalTime.Recorder()()
	txCount := len(this.Block.Transactions)
	txConflictErrors := make([]util.AtomicError, txCount)
	conflictDetector := conflict_detector.New(
		txCount*this.ConflictDetectorInboxPerTransaction,
		func(_ *conflict_detector.Operation, authors conflict_detector.Authors) {
			authors.Each(func(_ int, author conflict_detector.Author) {
				txConflictErrors[author.(vm2.TxId)].SetIfAbsent(errors.New(""))
			})
		})
	go conflictDetector.Run()
	defer conflictDetector.Halt()
	concurrent.Parallelize(this.NumConcurrentProcesses, txCount, func(int) func(int) {
		defer util.Recover(this.err.Catch())
		stateDB, stateDBCreateErr := state.New(this.BaseStateRoot, this.ReadDB)
		this.err.SetOrPanicIfPresent(stateDBCreateErr)
		return func(txId vm2.TxId) {
			errConflict := txConflictErrors[txId]
			defer util.Recover(errConflict.Catch())
			this.ExecuteTransaction(&base_vm.TransactionRequest{
				Transaction: this.Block.Transactions[txId],
				BlockHeader: &this.Block.BlockHeader,
				GasPool:     new(core.GasPool).AddGas(uint64(this.Block.GasLimit)),
				CheckNonce:  false,
				DB:          &OperationLoggingStateDB{stateDB, conflictDetector.NewLogger(txId)},
				OnEvmInstruction: func(pc uint64) (uint64, bool) {
					errConflict.PanicIfPresent()
					return pc, true
				},
				CanTransfer: func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
					return true
				},
			})
		}
	})
	defer this.metrics.ConflictPostProcessing.Recorder()()
	this.err.PanicIfPresent()
	conflictDetector.Halt()
	conflictingTx := conflictDetector.AwaitResult().Values()
	sort.Slice(conflictingTx, func(i, j int) bool {
		return conflictingTx[i].(vm2.TxId) < conflictingTx[j].(vm2.TxId)
	})
	this.result = &vm2.ConcurrentSchedule{vm2.NewTxIdSet(conflictingTx)}
}
