package state_transition

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/itr"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/emirpasic/gods/sets/treeset"
	"math/big"
)

type TaraxaEvm struct {
	externalApi     *api.ExternalApi
	chainConfig     *params.ChainConfig
	evmConfig       *vm.Config
	stateTransition *api.StateTransition
	db              state.Database
}

func (this *TaraxaEvm) generateSchedule() (result api.ConcurrentSchedule, err error) {
	var ERROR util.ErrorBarrier
	defer ERROR.Recover(func(e error) {
		err = e
	})
	txCount := len(this.stateTransition.Transactions)
	conflictDetector := new(conflict_tracking.ConflictDetector).Init(uint64(txCount * 60))
	go conflictDetector.Run()
	defer conflictDetector.SignalShutdown()

	barrier := make(chan interface{}, txCount)
	for txId := 0; txId < txCount; txId++ {
		txIds := itr.From(txId).Uint64()
		go func() {
			var ERROR util.ErrorBarrier
			defer func() {
				ERROR.Recover()
				barrier <- nil
			}()
			this.runMany(&RunParams{
				conflictDetector: conflictDetector,
				txIds: txIds,
				onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
					ERROR.CheckIn(result.dbErr, result.consensusErr)
					return true
				},
				onDone: func(db *state.StateDB, stateDBCreateErr error) {
					ERROR.CheckIn(stateDBCreateErr)
				},
				executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
					return func(pc uint64) (uint64, bool) {
						ERROR.CheckIn()
						if conflictDetector.IsCurrentlyInConflict(txId) {
							ERROR.CheckIn(errors.New("CONFLICT"))
						}
						return pc, true
					}
				},
			})
		}()
	}
	for i := 0; i < cap(barrier); i++ {
		<-barrier
		ERROR.CheckIn()
	}
	sequentialTx := treeset.NewWithIntComparator()
	for attempt := 0; true; attempt++ {
		util.Assert(attempt <= txCount, "Too many attempts")
		conflictDetector.SignalShutdown().Join()
		if !conflictDetector.HaveBeenConflicts() {
			break
		}
		conflictDetector.Reset(func(txId conflict_tracking.TxId) {
			util.Assert(!sequentialTx.Contains(txId), "Detected conflicts twice for : "+string(txId))
			sequentialTx.Add(txId)
			result.Sequential = append(result.Sequential, txId)
		})
		go conflictDetector.Run()
		this.runMany(&RunParams{
			conflictDetector: conflictDetector,
			txIds: itr.FromTreeSet(sequentialTx).Uint64(),
			onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
				ERROR.CheckIn(result.dbErr, result.consensusErr)
				return true
			},
			onDone: func(db *state.StateDB, stateDBCreateErr error) {
				ERROR.CheckIn(stateDBCreateErr)
			},
			executionControllerFactory: func(id conflict_tracking.TxId) vm.ExecutionController {
				return nil
			},
		})
	}
	return
}

func (this *TaraxaEvm) transitionState(schedule *api.ConcurrentSchedule) (ret api.StateTransitionResult, err error) {
	var ERROR util.ErrorBarrier
	defer ERROR.Recover(func(e error) {
		err = e
	})

	txCount := len(this.stateTransition.Transactions)
	conflictDetector := new(conflict_tracking.ConflictDetector).Init(uint64(txCount * 60))
	go conflictDetector.Run()
	defer conflictDetector.SignalShutdown()

	sequentialTx := treeset.NewWithIntComparator()
	// TODO non sync
	for _, txId := range schedule.Sequential {
		sequentialTx.Add(txId)
	}

	parallelCount := txCount - sequentialTx.Size()

	commitDb := state.NewDatabase(this.db.TrieDB().DiskDB().(ethdb.Database))
	stateDBChan := make(chan *state.StateDB, parallelCount)
	stateRootChan := make(chan common.Hash, 1)
	go func() {
		defer ERROR.Recover(func(error) {
			close(stateRootChan)
		})
		currentRoot := this.stateTransition.StateRoot
		for i := 0; i < cap(stateDBChan); i++ {
			stateDb := <-stateDBChan
			ERROR.CheckIn()
			rebaseErr := stateDb.Rebase(currentRoot, commitDb)
			ERROR.CheckIn(rebaseErr)
			root, commitErr := stateDb.Commit(true)
			ERROR.CheckIn(commitErr)
			currentRoot = root
		}
		stateRootChan <- currentRoot
	}()

	sequentialResultChan := make(chan *TransactionResult, sequentialTx.Size())
	go func() {
		defer ERROR.Recover(func(error) {
			close(sequentialResultChan)
			close(stateDBChan)
		})
		this.runMany(&RunParams{
			txIds:            itr.FromTreeSet(sequentialTx).Uint64(),
			conflictDetector: conflictDetector,
			onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
				ERROR.CheckIn(result.dbErr, result.consensusErr)
				sequentialResultChan <- result
				return true
			},
			onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
				ERROR.CheckIn(stateDBCreateErr)
				util.Try(func() {
					stateDBChan <- stateDB
				})
			},
			executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
				return func(pc uint64) (uint64, bool) {
					ERROR.CheckIn()
					if conflictDetector.HaveBeenConflicts() {
						ERROR.CheckIn(errors.New("CONFLICT"))
					}
					return pc, true
				}
			},
		})
	}()

	resultChans := make([]chan *TransactionResult, txCount)
	for txId := 0; txId < txCount; txId++ {
		if sequentialTx.Contains(txId) {
			resultChans[txId] = sequentialResultChan
			continue
		}
		resultChan := make(chan *TransactionResult, 1)
		resultChans[txId] = resultChan
		txId := txId
		go func() {
			defer ERROR.Recover(func(error) {
				close(resultChan)
				close(stateDBChan)
			})
			this.runMany(&RunParams{
				txIds:            itr.From(txId).Uint64(),
				conflictDetector: conflictDetector,
				onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
					ERROR.CheckIn(result.dbErr, result.consensusErr)
					resultChan <- result
					return true
				},
				onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
					ERROR.CheckIn(stateDBCreateErr)
					util.Try(func() {
						stateDBChan <- stateDB
					})
				},
				executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
					return func(pc uint64) (uint64, bool) {
						ERROR.CheckIn()
						if conflictDetector.HaveBeenConflicts() {
							ERROR.CheckIn(errors.New("CONFLICT"))
						}
						return pc, true
					}
				},
			})
		}()
	}

	gasPool := new(core.GasPool).AddGas(this.stateTransition.Block.GasLimit)
	for txId := 0; txId < txCount; txId++ {
		txResult := <-resultChans[txId]
		ERROR.CheckIn()
		txData := this.stateTransition.Transactions[txId]
		gasLimitReachedErr := gasPool.SubGas(txData.GasLimit)
		ERROR.CheckIn(gasLimitReachedErr)
		gasPool.AddGas(txData.GasLimit - txResult.gasUsed)
		ret.UsedGas += txResult.gasUsed
		ethReceipt := types.NewReceipt(nil, txResult.contractErr != nil, ret.UsedGas)
		if txData.To == nil {
			ethReceipt.ContractAddress = crypto.CreateAddress(txData.From, txData.Nonce)
		}
		ethReceipt.TxHash = txData.Hash;
		ethReceipt.GasUsed = txResult.gasUsed
		ethReceipt.Logs = txResult.logs
		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
		ret.Receipts = append(ret.Receipts, &api.TaraxaReceipt{
			ReturnValue:     txResult.value,
			ContractError:   txResult.contractErr,
			EthereumReceipt: ethReceipt,
		})
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}

	ret.StateRoot = <-stateRootChan
	ERROR.CheckIn()
	finalCommitErr := commitDb.TrieDB().Commit(ret.StateRoot, true)
	ERROR.CheckIn(finalCommitErr)
	return
}

func (this *TaraxaEvm) RunOne(
	conflictDetector *conflict_tracking.ConflictDetector,
	stateDB *state.StateDB,
	txId conflict_tracking.TxId,
	controller vm.ExecutionController,
) (
	*TransactionResult,
) {
	block := this.stateTransition.Block
	txData := this.stateTransition.Transactions[txId]
	gasPrice := api.BigInt(txData.GasPrice)
	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	txExecution := TransactionExecution{
		txId:      txId,
		txHash:    txData.Hash,
		blockHash: block.Hash,
		tx: types.NewMessage(
			txData.From, txData.To, txData.Nonce, api.BigInt(txData.Amount),
			txData.GasLimit, gasPrice, *txData.Data,
			true,
		),
		evmContext: &vm.Context{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     this.externalApi.GetHeaderHashByBlockNumber,
			Origin:      txData.From,
			Coinbase:    block.Coinbase,
			BlockNumber: api.BigInt(block.Number),
			Time:        api.BigInt(block.Time),
			Difficulty:  api.BigInt(block.Difficulty),
			GasLimit:    block.GasLimit,
			GasPrice:    new(big.Int).Set(gasPrice),
		},
		chainConfig: this.chainConfig,
		evmConfig:   this.evmConfig,
	}
	return txExecution.Run(&TransactionParams{
		stateDB:       stateDB,
		conflicts:     conflictDetector,
		gasPool:       gasPool,
		executionCtrl: controller,
	})
}

// TODO separate class
type RunParams struct {
	conflictDetector           *conflict_tracking.ConflictDetector
	txIds                      itr.IntIterator
	executionControllerFactory func(conflict_tracking.TxId) vm.ExecutionController
	onTxResult                 func(conflict_tracking.TxId, *TransactionResult) bool
	onDone                     func(*state.StateDB, error)
}

func (this *TaraxaEvm) runMany(args *RunParams) {
	stateDB, stateDbCreateErr := state.New(this.stateTransition.StateRoot, this.db)
	defer args.onDone(stateDB, stateDbCreateErr)
	if (stateDbCreateErr != nil) {
		return
	}
	for txId, done := args.txIds(); !done; txId, done = args.txIds() {
		result := this.RunOne(args.conflictDetector, stateDB, txId, args.executionControllerFactory(txId))
		if !args.onTxResult(txId, result) {
			break
		}
	}
}
