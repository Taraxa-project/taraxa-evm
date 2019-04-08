package state_transition

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/itr"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
)

type TaraxaEvm struct {
	externalApi     *api.ExternalApi
	chainConfig     *params.ChainConfig
	evmConfig       *vm.Config
	stateTransition *api.StateTransition
	db              state.Database
}

func (this *TaraxaEvm) generateSchedule() (result api.ConcurrentSchedule, err error) {
	var fatalErr util.SharedError
	defer fatalErr.Recover(func(e error) {
		err = e
	})
	conflictDetector := new(conflict_tracking.ConflictDetector).Init()
	commonRunParams := RunParams{
		conflictDetector: conflictDetector,
		onTxResult: func(conflict_tracking.TxId, result *TransactionResult) bool {
			fatalErr.CheckIn(result.dbErr, result.consensusErr)
			return true
		},
		onDone: func(*state.StateDB, stateDBCreateErr error) {
			fatalErr.CheckIn(stateDBCreateErr)
		},
		executionControllerFactory: func(id conflict_tracking.TxId) vm.ExecutionController {
			return ni
		},
	}
	txCount := len(this.stateTransition.Transactions)
	barrier := make(chan interface{}, txCount)
	for txId := conflict_tracking.TxId(0); txId < txCount; txId++ {
		go func() {
			var conflictError util.SharedError
			defer func() {
				fatalErr.Recover()
				conflictError.Recover()
				barrier <- nil
			}()
			runParams := RunParams(commonRunParams)
			runParams.txIds = itr.From(txId).Uint64()
			runParams.executionControllerFactory = func(txId conflict_tracking.TxId) vm.ExecutionController {
				return func(pc uint64) (uint64, bool) {
					fatalErr.CheckIn()
					if conflictDetector.InConflict(txId) {
						conflictError.CheckIn(errors.New("CONFLICT"))
					}
					return pc, true
				}
			}
			this.runMany(&runParams)
		}()
	}
	for i := 0; i < cap(barrier); i++ {
		<-barrier
		fatalErr.CheckIn()
	}
	sequentialTx := treeset.NewWith(utils.UInt64Comparator)
	for attempt := 0; true; attempt++ {
		util.Assert(attempt <= txCount, "Too many attempts")
		conflictingTxIds := conflictDetector.Reset()
		if (len(conflictingTxIds) == 0) {
			break
		}
		for _, txId := range conflictingTxIds {
			util.Assert(!sequentialTx.Contains(txId), "Detected conflicts twice for : "+txId)
			sequentialTx.Add(txId)
			result.Sequential = append(result.Sequential, txId)
		}
		runParams := RunParams(commonRunParams)
		runParams.txIds = itr.FromTreeSet(sequentialTx).Uint64()
		this.runMany(&runParams)
	}
	return
}

func (this *TaraxaEvm) transitionState(schedule *api.ConcurrentSchedule) (ret api.StateTransitionResult, err error) {
	var sharedErr util.SharedError
	defer sharedErr.Recover(func(e error) {
		err = e
	})

	conflictDetector := new(conflict_tracking.ConflictDetector)

	commonRunParams := RunParams{
		conflictDetector: conflictDetector,
		onTxResult: func(conflict_tracking.TxId, result *TransactionResult) bool {
			fatalErr.CheckIn(result.dbErr, result.consensusErr)
			return true
		},
		onDone: func(*state.StateDB, stateDBCreateErr error) {
			fatalErr.CheckIn(stateDBCreateErr)
		},
		executionControllerFactory: func(id conflict_tracking.TxId) vm.ExecutionController {
			return ni
		},
	}

	sequentialTx := treeset.NewWith(utils.UInt64Comparator, schedule.Sequential...)
	txCount := len(this.stateTransition.Transactions)
	parallelCount := txCount - sequentialTx.Size()

	commitDb := state.NewDatabase(this.db.TrieDB().DiskDB())
	stateDBChan := make(chan *state.StateDB, parallelCount)
	stateRootChan := make(chan common.Hash, 1)
	go func() {
		defer sharedErr.Recover(func(error) {
			close(stateRootChan)
		})
		currentRoot := this.stateTransition.StateRoot
		for _ := 0; _ < cap(stateDBChan); _++ {
			// DEADLOCK, need to close statedb chan
			stateDb := <-stateDBChan
			rebaseErr := stateDb.Rebase(currentRoot, commitDb)
			sharedErr.CheckIn(rebaseErr)
			root, commitErr := stateDb.Commit(true)
			sharedErr.CheckIn(commitErr)
			currentRoot = root
		}
		stateRootChan <- currentRoot
	}()

	sequentialResultChan := make(chan *TransactionResult, sequentialTx.Size())
	go func() {
		defer sharedErr.Recover(func(error) {
			close(sequentialResultChan)
		})
		this.runMany(&RunParams{
			txIds:            itr.FromTreeSet(sequentialTx),
			conflictDetector: conflictDetector,
			onTxResult: func(conflict_tracking.TxId, result *TransactionResult) bool {
				sharedErr.CheckIn(result.dbErr, result.consensusErr)
				sequentialResultChan <- result
				return true
			},
			onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
				sharedErr.CheckIn(stateDBCreateErr)
				stateDBChan <- stateDB
			},
			executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
				return func(pc uint64) (uint64, bool) {
					sharedErr.CheckIn()
					if conflictDetector.InConflict(txId) {
						sharedErr.CheckIn(errors.New("CONFLICT"))
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
			defer sharedErr.Recover(func(error) {
				close(resultChan)
			})
			this.runMany(&RunParams{
				txIds:            itr.From(txId),
				conflictDetector: conflictDetector,
				onTxResult: func(conflict_tracking.TxId, result *TransactionResult) bool {
					sharedErr.CheckIn(result.dbErr, result.consensusErr)
					resultChan <- result
					return true
				},
				onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
					sharedErr.CheckIn(stateDBCreateErr)
					stateDBChan <- stateDB
				},
				func(txId conflict_tracking.TxId) vm.ExecutionController {
					return func(pc uint64) (uint64, bool) {
						sharedErr.CheckIn()
						if conflictDetector.InConflict(txId) {
							sharedErr.CheckIn(errors.New("CONFLICT"))
						}
						return pc, true
					}
				},
			})
		}()
	}

	for txId := 0; txId < txCount; txId++ {
		txResult := <-resultChans[txId]
		sharedErr.CheckIn()
		txData := this.stateTransition.Transactions[txId]
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

	ret.StateRoot <- stateRootChan
	sharedErr.CheckIn()
	finalCommitErr := commitDb.TrieDB().Commit(ret.StateRoot)
	sharedErr.CheckIn(finalCommitErr)
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
		evmContext: vm.Context{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     this.externalApi.GetHeaderHashByBlockNumber,
			Origin:      tx.From(),
			Coinbase:    block.Coinbase,
			BlockNumber: api.BigInt(block.Number),
			Time:        api.BigInt(block.Time),
			Difficulty:  api.BigInt(block.Difficulty),
			GasLimit:    api.block.GasLimit,
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

type RunParams struct {
	conflictDetector           *conflict_tracking.ConflictDetector
	txIds                      itr.Uint64Iterator
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
		result := this.RunOne(detector, stateDB, txId, args.executionControllerFactory(txId))
		if !args.onTxResult(result) {
			break
		}
	}
}
