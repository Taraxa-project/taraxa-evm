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
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/state_db"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/barrier"
	"github.com/Taraxa-project/taraxa-evm/main/util/itr"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/emirpasic/gods/sets/treeset"
	"math/big"
)

var all_transactions conflict_detector.Author = "ALL_TRANSACTIONS"
var sequential_group conflict_detector.Author = "SEQUENTIAL_GROUP"

type TaraxaEvm struct {
	externalApi     *api.ExternalApi
	chainConfig     *params.ChainConfig
	evmConfig       *vm.Config
	stateTransition *api.StateTransition
	readDB          state.Database
}

func (this *TaraxaEvm) generateSchedule() (result api.ConcurrentSchedule, err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	txCount := len(this.stateTransition.Transactions)
	conflictDetector := new(conflict_detector.ConflictDetector).Init(txCount * 60)
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	parallelRoundDone := barrier.New(txCount)
	for txId := 0; txId < txCount; txId++ {
		txId := txId
		go func() {
			var errConflict, errTxExecution util.ErrorBarrier
			defer util.Recover(
				errFatal.Catch(),
				errConflict.Catch(),
				errTxExecution.Catch(func(e error) {
					errFatal.SetIfAbsent(e)
				}),
			)
			defer parallelRoundDone.CheckIn()
			this.runTransactions(&RunParams{
				conflictDetector: conflictDetector,
				conflictAuthor:   txId,
				txIds:            itr.From(txId).Int(),
				onTxResult: func(txId api.TxId, result *TransactionResult) bool {
					errTxExecution.CheckIn(result.dbErr, result.consensusErr)
					return true
				},
				onDone: func(db *state.StateDB, stateDBCreateErr error) {
					errFatal.CheckIn(stateDBCreateErr)
				},
				executionControllerFactory: func(txId api.TxId) vm.ExecutionController {
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
	conflictingAuthors := conflictDetector.RequestShutdown().Reset()
	sequentialSet := treeset.NewWithIntComparator()
	conflictingAuthors.Each(func(index int, author conflict_detector.Author) {
		if author == all_transactions {
			return
		}
		sequentialSet.Add(author)
	})
	result.Sequential = make([]api.TxId, sequentialSet.Size())
	sequentialSet.Each(func(index int, value interface{}) {
		result.Sequential[index] = value.(api.TxId)
	})
	return
}

func (this *TaraxaEvm) transitionState(schedule *api.ConcurrentSchedule) (ret api.StateTransitionResult, err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	txCount := len(this.stateTransition.Transactions)
	// TODO non sync
	sequentialTx := treeset.NewWithIntComparator()
	for _, txId := range schedule.Sequential {
		sequentialTx.Add(txId)
	}
	parallelTxCount := txCount - sequentialTx.Size()
	conflictDetector := new(conflict_detector.ConflictDetector).Init(txCount * 60)
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()

	intermediateStateDbChan := make(chan *state.StateDB, parallelTxCount+1)
	finalStateDbChan := make(chan *state.StateDB, 1)
	go func() {
		defer util.Recover(errFatal.Catch(func(err error) {
			close(finalStateDbChan)
		}))
		diskDb := this.readDB.TrieDB().DiskDB().(ethdb.Database)
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
	resultChans := make([]chan *TransactionResult, txCount)
	parallelRoundDone := barrier.New(parallelTxCount)
	for txId := 0; txId < txCount; txId++ {
		if sequentialTx.Contains(txId) {
			resultChans[txId] = sequentialResultChan
			continue
		}
		resultChan := make(chan *TransactionResult, 1)
		resultChans[txId] = resultChan
		txId := txId
		go func() {
			defer util.Recover(errFatal.Catch(func(error) {
				close(resultChan)
				close(intermediateStateDbChan)
			}))
			defer parallelRoundDone.CheckIn()
			this.runTransactions(&RunParams{
				conflictAuthor:   txId,
				txIds:            itr.From(txId).Int(),
				conflictDetector: conflictDetector,
				onTxResult: func(txId api.TxId, result *TransactionResult) bool {
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
				executionControllerFactory: func(txId api.TxId) vm.ExecutionController {
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

	go func() {
		defer util.Recover(errFatal.Catch(func(error) {
			close(sequentialResultChan)
			close(intermediateStateDbChan)
		}))
		parallelRoundDone.Await()
		errFatal.CheckIn()
		this.runTransactions(&RunParams{
			conflictAuthor:   sequential_group,
			txIds:            itr.FromTreeSet(sequentialTx).Int(),
			conflictDetector: conflictDetector,
			onTxResult: func(txId api.TxId, result *TransactionResult) bool {
				errFatal.CheckIn(result.dbErr, result.consensusErr)
				sequentialResultChan <- result
				return true
			},
			onDone: func(stateDB *state.StateDB, stateDBCreateErr error) {
				errFatal.CheckIn(stateDBCreateErr)
				intermediateStateDbChan <- stateDB
			},
			executionControllerFactory: func(txId api.TxId) vm.ExecutionController {
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

	transientStateDB, stateDbCreateErr := state.New(this.stateTransition.StateRoot, this.readDB)
	errFatal.CheckIn(stateDbCreateErr)
	gasPool := new(core.GasPool).AddGas(this.stateTransition.Block.GasLimit)
	for txId := 0; txId < txCount; txId++ {
		txResult := <-resultChans[txId]
		errFatal.CheckIn()

		txData := this.stateTransition.Transactions[txId]

		nonceErr := core.CheckNonce(transientStateDB, txData.From, txData.Nonce)
		errFatal.CheckIn(nonceErr)

		gasLimitReachedErr := gasPool.SubGas(txData.GasLimit)
		errFatal.CheckIn(gasLimitReachedErr)
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
		for account, balanceDelta := range txResult.transientState.BalanceDeltas {
			if balanceDelta.Sign() != 0 {
				transientStateDB.AddBalance(account, balanceDelta)
				if transientStateDB.GetBalance(account).Sign() < 0 {
					// TODO record and replay validation events
					errFatal.CheckIn(vm.ErrInsufficientBalance)
				}
			}
		}
		for account, nonceDelta := range txResult.transientState.NonceDeltas {
			transientStateDB.AddNonce(account, nonceDelta)
		}
	}
	// TODO
	//intermediateStateDbChan <- transientStateDB

	finalStateDb := <-finalStateDbChan
	errFatal.CheckIn()

	finalRoot, commitErr := finalStateDb.Commit(true)
	errFatal.CheckIn(commitErr)

	finalCommitErr := finalStateDb.Database().TrieDB().Commit(finalRoot, true)
	errFatal.CheckIn(finalCommitErr)

	conflictingAuthors := conflictDetector.RequestShutdown().Reset()
	if !conflictingAuthors.Empty() {
		errFatal.CheckIn(errors.New("CONFLICTS: " + conflictingAuthors.String()))
	}
	ret.StateRoot = finalRoot
	return
}

func (this *TaraxaEvm) RunOne(
	taraxaDB *state_db.TaraxaStateDB,
	txId api.TxId,
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
			false,
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
		taraxaDb:      taraxaDB,
		gasPool:       gasPool,
		executionCtrl: controller,
	})
}

// TODO separate class
type RunParams struct {
	conflictDetector           *conflict_detector.ConflictDetector
	conflictAuthor             conflict_detector.Author
	txIds                      itr.IntIterator
	executionControllerFactory func(api.TxId) vm.ExecutionController
	onTxResult                 func(api.TxId, *TransactionResult) bool
	onDone                     func(*state.StateDB, error)
}

func (this *TaraxaEvm) runTransactions(args *RunParams) {
	stateDB, stateDbCreateErr := state.New(this.stateTransition.StateRoot, this.readDB)
	defer args.onDone(stateDB, stateDbCreateErr)
	if (stateDbCreateErr != nil) {
		return
	}
	taraxaDb := state_db.NewDB(stateDB, args.conflictDetector.NewLogger(args.conflictAuthor))
	if args.executionControllerFactory == nil {
		args.executionControllerFactory = func(id api.TxId) vm.ExecutionController {
			return nil
		}
	}
	if (stateDbCreateErr == nil) {
		for txId, done := args.txIds(); !done; txId, done = args.txIds() {
			result := this.RunOne(taraxaDb, txId, args.executionControllerFactory(txId))
			if !args.onTxResult(txId, result) {
				break
			}
		}
	}
}
