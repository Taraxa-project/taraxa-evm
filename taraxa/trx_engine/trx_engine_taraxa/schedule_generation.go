package trx_engine_taraxa

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_taraxa/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/internal/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

var conflictError = errors.New("")

type scheduleGeneration struct {
	*TaraxaTrxEngine
	*trx_engine.StateTransitionRequest
	err concurrent.AtomicError
}

func newScheduleGeneration(vm *TaraxaTrxEngine, req *trx_engine.StateTransitionRequest) *scheduleGeneration {
	return &scheduleGeneration{TaraxaTrxEngine: vm, StateTransitionRequest: req}
}

func (this *scheduleGeneration) run() (ret *trx_engine.ConcurrentSchedule, err error) {
	txCount := len(this.Block.Transactions)
	txConflictErrors := make([]concurrent.AtomicError, txCount)
	sendConflictErrorToTx := func(author conflict_detector.Author) {
		txConflictErrors[author.(trx_engine.TxIndex)].SetIfAbsent(conflictError)
	}
	conflictDetectorActor := conflict_detector.NewConflictDetectorActor(
		conflict_detector.NewConflictDetector(),
		txCount*this.ConflictDetectorInboxPerTransaction,
		func(op *conflict_detector.Operation, conflicts *conflict_detector.AuthorsByOperation, hasCaused bool) {
			sendConflictErrorToTx(op.Author)
			if !hasCaused {
				return
			}
			for _, authors := range conflicts {
				for author := range authors {
					sendConflictErrorToTx(author)
				}
			}
		})
	this.err.AddHandler(func(_ error) {
		conflictDetectorActor.ForceShutdown()
	})
	go conflictDetectorActor.Run()
	concurrent.Parallelize(this.NumConcurrentProcesses, txCount, func(int) concurrent.IterationController {
		return func(txIndex trx_engine.TxIndex) (shouldExit bool) {
			if this.err.IsPresent() {
				return true
			}
			stateDB, stateDBCreateErr := state.New(this.BaseStateRoot, this.ReadDB)
			if this.err.SetIfAbsent(stateDBCreateErr) {
				return true
			}
			errConflict := txConflictErrors[txIndex]
			defer errConflict.Recover()
			tx := this.Block.Transactions[txIndex]
			stateDB.SetNonce(tx.From, uint64(tx.Nonce))
			result := this.ExecuteTransaction(&trx_engine_base.TransactionRequest{
				Transaction: tx,
				BlockHeader: &this.Block.BlockHeader,
				GasPool:     new(core.GasPool).AddGas(uint64(tx.Gas)),
				DB: &StateDBForConflictDetection{
					StateDB:      stateDB,
					LogOperation: conflictDetectorActor.NewOperationLogger(txIndex),
				},
				OnEvmInstruction: func(pc uint64) (uint64, bool) {
					errConflict.PanicIfPresent()
					return pc, true
				},
			})
			return this.err.SetIfAbsent(result.ConsensusErr)
		}
	})
	conflictDetector := conflictDetectorActor.SendTerminator().Await()
	if err = this.err.Get(); err != nil {
		return
	}
	this.err.PanicIfPresent()
	ret = new(trx_engine.ConcurrentSchedule)
	for txIndex := range this.Block.Transactions {
		if conflictDetector.AuthorsInConflict[txIndex] {
			ret.SequentialTransactions = append(ret.SequentialTransactions, txIndex)
		}
	}
	return
}
