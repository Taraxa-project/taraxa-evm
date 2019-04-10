package state_transition

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/barrier"
	"github.com/Taraxa-project/taraxa-evm/main/util/itr"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/emirpasic/gods/sets/treeset"
	"math/big"
)

const all_tx_conflict_detector_author = "ALL_TRANSACTIONS"

type TaraxaEvm struct {
	externalApi     *api.ExternalApi
	chainConfig     *params.ChainConfig
	evmConfig       *vm.Config
	stateTransition *api.StateTransition
	db              state.Database
}

func (this *TaraxaEvm) generateSchedule() (result api.ConcurrentSchedule, err error) {
	var errFatal, errTxExecution util.ErrorBarrier
	txCount := len(this.stateTransition.Transactions)
	conflictDetector := new(conflict_tracking.ConflictDetector).Init(uint64(txCount * 60))
	parallelRoundDone := barrier.New(txCount)
	defer errFatal.Recover(func(e error) {
		err = e
	})
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	conflictDetector.Submit(&conflict_tracking.Operation{
		IsWrite: true,
		Author:  all_tx_conflict_detector_author,
		Key:     this.stateTransition.Block.Coinbase.Hex(),
	})
	for txId := 0; txId < txCount; txId++ {
		txIds := itr.From(txId).Int()
		go func() {
			var errConflict, errTxExecutionLocal util.ErrorBarrier
			defer errConflict.Recover()
			defer errTxExecutionLocal.Recover(func(e error) {
				errTxExecution.SetIfAbsent(e)
			})
			defer errFatal.Recover()
			defer parallelRoundDone.CheckIn()
			this.runTransactions(&RunParams{
				conflictDetector: conflictDetector,
				txIds:            txIds,
				onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
					errTxExecutionLocal.CheckIn(result.dbErr, result.consensusErr)
					return true
				},
				onDone: func(db *state.StateDB, stateDBCreateErr error) {
					errFatal.CheckIn(stateDBCreateErr)
				},
				executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
					return func(pc uint64) (uint64, bool) {
						errFatal.CheckIn()
						if conflictDetector.IsCurrentlyInConflict(txId) {
							errConflict.CheckIn(errors.New("CONFLICT"))
						}
						return pc, true
					}
				},
			})
		}()
	}
	parallelRoundDone.Await()
	errFatal.CheckIn()
	sequentialTx := treeset.NewWithIntComparator()
	for attempt := 0; true; attempt++ {
		util.Assert(attempt <= txCount, "Too many attempts")
		conflictDetector.RequestShutdown().Join()
		if !conflictDetector.HaveBeenConflicts() {
			err := errTxExecution.Get()
			errFatal.CheckIn(err)
			break
		}
		errTxExecution = util.ErrorBarrier{}
		conflictDetector.Reset(func(author interface{}) {
			if author == all_tx_conflict_detector_author {
				return
			}
			txId := author.(conflict_tracking.TxId)
			util.Assert(!sequentialTx.Contains(txId), "Detected conflicts twice for : "+string(txId))
			sequentialTx.Add(txId)
			result.Sequential = append(result.Sequential, txId)
		})
		go conflictDetector.Run()
		func() {
			defer errTxExecution.Recover()
			this.runTransactions(&RunParams{
				conflictDetector: conflictDetector,
				txIds:            itr.FromTreeSet(sequentialTx).Int(),
				onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
					errTxExecution.CheckIn(result.dbErr, result.consensusErr)
					return true
				},
				onDone: func(db *state.StateDB, stateDBCreateErr error) {
					errFatal.CheckIn(stateDBCreateErr)
				},
			})
		}()
	}
	return
}

func (this *TaraxaEvm) transitionState(schedule *api.ConcurrentSchedule) (ret api.StateTransitionResult, err error) {
	var errFatal util.ErrorBarrier
	defer errFatal.Recover(func(e error) {
		err = e
	})

	txCount := len(this.stateTransition.Transactions)
	conflictDetector := new(conflict_tracking.ConflictDetector).Init(uint64(txCount * 60))
	// TODO non sync
	sequentialTx := treeset.NewWithIntComparator()
	for _, txId := range schedule.Sequential {
		sequentialTx.Add(txId)
		conflictDetector.IgnoreAuthor(txId)
	}
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	conflictDetector.Submit(&conflict_tracking.Operation{
		IsWrite: true,
		Author:  all_tx_conflict_detector_author,
		Key:     this.stateTransition.Block.Coinbase.Hex(),
	})

	intermediateStateDbChan := make(chan *state.StateDB, txCount-sequentialTx.Size()+1)
	finalStateDbChan := make(chan *state.StateDB, 1)
	go func() {
		defer errFatal.Recover(func(err error) {
			close(finalStateDbChan)
		})
		diskDb := this.db.TrieDB().DiskDB().(ethdb.Database)
		commitDb := state.NewDatabase(diskDb)
		currentRoot := this.stateTransition.StateRoot
		var currentStateDB *state.StateDB
		for i := 0; i < cap(intermediateStateDbChan); i++ {
			currentStateDB = <-intermediateStateDbChan
			errFatal.CheckIn()

			rebaseErr := currentStateDB.Rebase(currentRoot, commitDb)
			errFatal.CheckIn(rebaseErr)

			root, commitErr := currentStateDB.Commit(true)
			errFatal.CheckIn(commitErr)
			currentRoot = root
		}
		finalStateDbChan <- currentStateDB
	}()

	sequentialResultChan := make(chan *TransactionResult, sequentialTx.Size())
	go func() {
		defer errFatal.Recover(func(error) {
			close(sequentialResultChan)
			close(intermediateStateDbChan)
		})
		this.runTransactions(&RunParams{
			txIds:            itr.FromTreeSet(sequentialTx).Int(),
			conflictDetector: conflictDetector,
			onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
				errFatal.CheckIn(result.dbErr, result.consensusErr)
				sequentialResultChan <- result
				return true
			},
			onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
				errFatal.CheckIn(stateDBCreateErr)
				util.Try(func() {
					intermediateStateDbChan <- stateDB
				})
			},
			executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
				return func(pc uint64) (uint64, bool) {
					errFatal.CheckIn()
					if conflictDetector.HaveBeenConflicts() {
						errFatal.CheckIn(errors.New("CONFLICT"))
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
		txIds := itr.From(txId).Int()
		go func() {
			defer errFatal.Recover(func(error) {
				close(resultChan)
				close(intermediateStateDbChan)
			})
			this.runTransactions(&RunParams{
				txIds:            txIds,
				conflictDetector: conflictDetector,
				onTxResult: func(txId conflict_tracking.TxId, result *TransactionResult) bool {
					errFatal.CheckIn(result.dbErr, result.consensusErr)
					resultChan <- result
					return true
				},
				onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
					errFatal.CheckIn(stateDBCreateErr)
					util.Try(func() {
						intermediateStateDbChan <- stateDB
					})
				},
				executionControllerFactory: func(txId conflict_tracking.TxId) vm.ExecutionController {
					return func(pc uint64) (uint64, bool) {
						errFatal.CheckIn()
						if conflictDetector.HaveBeenConflicts() {
							errFatal.CheckIn(errors.New("CONFLICT"))
						}
						return pc, true
					}
				},
			})
		}()
	}

	gasPool := new(core.GasPool).AddGas(this.stateTransition.Block.GasLimit)
	beneficiaryReward := big.NewInt(0)
	for txId := 0; txId < txCount; txId++ {
		txResult := <-resultChans[txId]
		errFatal.CheckIn()
		txData := this.stateTransition.Transactions[txId]
		gasLimitReachedErr := gasPool.SubGas(txData.GasLimit)
		errFatal.CheckIn(gasLimitReachedErr)
		gasPool.AddGas(txData.GasLimit - txResult.gasUsed)

		gasFee := new(big.Int).Mul(new(big.Int).SetUint64(txResult.gasUsed), api.BigInt(txData.GasPrice))
		beneficiaryReward.Add(beneficiaryReward, gasFee)

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

	conflictDetector.RequestShutdown()

	finalStateDb := <-finalStateDbChan
	errFatal.CheckIn()

	finalStateDb.AddBalance(this.stateTransition.Block.Coinbase, beneficiaryReward)
	finalRoot, commitErr := finalStateDb.Commit(true)
	errFatal.CheckIn(commitErr)

	finalCommitErr := finalStateDb.Database().TrieDB().Commit(finalRoot, true)
	errFatal.CheckIn(finalCommitErr)

	conflictDetector.Join()
	if conflictDetector.HaveBeenConflicts() {
		errFatal.CheckIn(errors.New("CONFLICT"))
	}
	ret.StateRoot = finalRoot
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

func (this *TaraxaEvm) runTransactions(args *RunParams) {
	stateDB, stateDbCreateErr := state.New(this.stateTransition.StateRoot, this.db)
	defer args.onDone(stateDB, stateDbCreateErr)
	if (stateDbCreateErr != nil) {
		return
	}
	if args.executionControllerFactory == nil {
		args.executionControllerFactory = func(id conflict_tracking.TxId) vm.ExecutionController {
			return nil
		}
	}
	if (stateDbCreateErr == nil) {
		for txId, done := args.txIds(); !done; txId, done = args.txIds() {
			result := this.RunOne(args.conflictDetector, stateDB, txId, args.executionControllerFactory(txId))
			if !args.onTxResult(txId, result) {
				break
			}
		}
	}
}
