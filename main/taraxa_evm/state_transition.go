package taraxa_evm

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_state_db"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type stateTransition struct {
	*TaraxaVM
	*api.StateTransitionRequest
	*api.ConcurrentSchedule
	metrics          Metrics
	conflictDetector *conflict_detector.ConflictDetector
	opLoggerFactory  conflict_detector.OperationLoggerFactory
	err              util.ErrorBarrier
}

type parallelTxResult struct {
	txId        api.TxId
	receipt     *api.TaraxaReceipt
	stateChange *taraxa_state_db.StateChange
}

func (this *stateTransition) init() {
	this.opLoggerFactory = conflict_detector.NoopLoggerFactory
	txCount := len(this.Block.Transactions)
	if txCount == this.SequentialTransactions.Size() {
		return
	}
	this.conflictDetector = conflict_detector.New((txCount + 1) * 60)
}

func (this stateTransition) Run() (ret *api.StateTransitionResult, err error) {
	ret = new(api.StateTransitionResult)
	defer util.Recover(this.err.Catch(util.SetTo(&err)))
	block := this.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		ret.StateRoot = this.applyGenesisBlock()
		return
	}
	this.init()

	txCount, sequentialTxCount := len(this.Block.Transactions), this.SequentialTransactions.Size()

	parallelResults := make(chan *parallelTxResult, txCount-sequentialTxCount)
	for txId := 0; txId < txCount; txId++ {
		if this.SequentialTransactions.Contains(txId) {
			continue
		}
		panic("don't go here")
		txId := txId
		go func() {
			defer util.Recover(this.err.Catch(func(error) {
				util.Try(func() {
					close(parallelResults)
				})
			}))
			ethereumStateDB := this.newStateDB()
			taraxaDb := this.newTaraxaStateDB(ethereumStateDB, txId)
			gasPool := new(core.GasPool).AddGas(math.MaxUint64)
			receipt := this.executeTransaction(txId, taraxaDb, gasPool, false)
			util.Try(func() {
				parallelResults <- &parallelTxResult{txId, receipt, taraxaDb.CommitLocally()}
			})
		}()
	}

	stateDB := this.newStateDB()

	this.applyForks(stateDB)

	//txResults := make([]*transactionResult, txCount)
	//for i := 0; i < cap(parallelResults); i++ {
	//	result := <-parallelResults
	//	this.err.CheckIn()
	//	stateDB.MergeChanges(result.stateChange.StateChange)
	//	txResults[result.txId] = result.txResult
	//}

	//taraxaDB := this.newTaraxaStateDB(stateDB, sequential_group)
	//this.SequentialTransactions.Each(func(_ int, value interface{}) {
	//	txId := value.(api.TxId)
	//	gasPool := new(core.GasPool).AddGas(math.MaxUint64)
	//	result := this.executeTransaction(txId, taraxaDB, gasPool, false)
	//	txResults[txId] = result
	//})

	//gasPool := new(core.GasPool).AddGas(block.GasLimit)
	//for txId, txResult := range txResults {
	//	txData := transactions[txId]
	//
	//	nonceErr := core.CheckNonce(stateDB, txData.From, txData.Nonce)
	//	this.err.CheckIn(nonceErr)
	//
	//	blockGasDepletedErr := gasPool.SubGas(txData.GasLimit)
	//	this.err.CheckIn(blockGasDepletedErr)
	//	gasPool.AddGas(txData.GasLimit - txResult.gasUsed)
	//
	//	for account, nonceDelta := range txResult.transientState.NonceDeltas {
	//		if !stateDB.Exist(account) {
	//			panic("account doesn't exist", account.Hex())
	//		}
	//		stateDB.AddNonce(account, nonceDelta)
	//	}
	//	for account, balanceDelta := range txResult.transientState.BalanceDeltas {
	//		if !stateDB.Exist(account) {
	//			panic("account doesn't exist", account.Hex())
	//		}
	//		//if balanceDelta.Sign() == 0 {
	//		//	continue
	//		//}
	//		stateDB.AddBalance(account, balanceDelta)
	//		if balanceDelta < 0 && stateDB.GetBalance(account).Sign() < 0 {
	//			// TODO record and replay validation events
	//			this.err.CheckIn(vm.ErrInsufficientBalance)
	//		}
	//	}
	//
	//	ret.UsedGas += txResult.gasUsed
	//	ethReceipt := types.NewReceipt(txResult.rootBytes, txResult.contractErr != nil, ret.UsedGas)
	//	if txData.To == nil {
	//		ethReceipt.ContractAddress = crypto.CreateAddress(txData.From, txData.Nonce)
	//	}
	//	ethReceipt.TxHash = txData.Hash;
	//	ethReceipt.GasUsed = txResult.gasUsed
	//	ethReceipt.Logs = txResult.logs
	//	ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
	//	ret.Receipts = append(ret.Receipts, &api.TaraxaReceipt{
	//		ReturnValue:     txResult.value,
	//		ContractError:   txResult.contractErr,
	//		EthereumReceipt: ethReceipt,
	//	})
	//	ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	//}

	this.applyBlockRewards(stateDB)

	//util.Assert(finalRoot == this.ExpectedRoot)
	//this.err.CheckIn(this.PersistentCommit(finalRoot))
	return
}

func (this *stateTransition) executeTransaction(txId api.TxId, db StateDB, gp *core.GasPool, checkNonce bool) *api.TaraxaReceipt {
	block := this.Block
	tx := block.Transactions[txId]
	result := this.TaraxaVM.executeTransaction(&transactionRequest{
		txId, tx, &block.BlockHeader, db, this.onEvmInstruction, gp, checkNonce,
	})
	this.err.CheckIn(result.consensusErr)
	ethReceipt := types.NewReceipt(nil, result.vmErr != nil, 0)
	if tx.To == nil {
		ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, tx.Nonce)
	}
	ethReceipt.TxHash = tx.Hash;
	ethReceipt.GasUsed = result.gasUsed
	ethReceipt.Logs = db.GetLogs(tx.Hash)
	ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
	return &api.TaraxaReceipt{
		ReturnValue:     result.vmReturnValue,
		ContractError:   result.vmErr,
		EthereumReceipt: ethReceipt,
	}
}

func (this *stateTransition) tryReportConflicts() {
	conflictingAuthors := this.conflictDetector.RequestShutdown().Reset()
	if !conflictingAuthors.Empty() {
		this.err.CheckIn(errors.New("Conflicts detected: " + conflictingAuthors.String()))
	}
}

func (this *stateTransition) onEvmInstruction(programCounter uint64) (programCounterChanged uint64, canProceed bool) {
	if this.conflictDetector != nil {
		this.err.CheckIn()
		if this.conflictDetector.HaveBeenConflicts() {
			this.tryReportConflicts()
		}
	}
	return programCounter, true
}

func (this *stateTransition) applyGenesisBlock() common.Hash {
	_, _, genesisSetupErr := core.SetupGenesisBlock(this.WriteDB, this.Genesis)
	this.err.CheckIn(genesisSetupErr)
	return this.Genesis.ToBlock(nil).Root()
}

func (this *stateTransition) applyForks(stateDB *state.StateDB) {
	chainConfig := this.Genesis.Config
	DAOForkBlock := chainConfig.DAOForkBlock
	if chainConfig.DAOForkSupport && DAOForkBlock != nil && DAOForkBlock.Cmp(this.Block.Number) == 0 {
		misc.ApplyDAOHardFork(stateDB)
	}
}

func (this *stateTransition) applyBlockRewards(stateDB *state.StateDB) {
	ethash.AccumulateRewards(this.Genesis.Config, stateDB, &this.Block.HeaderNumerAndCoinbase, this.Block.Uncles...)
}

func (this *stateTransition) newTaraxaStateDB(db *state.StateDB, author conflict_detector.Author) *taraxa_state_db.TaraxaStateDB {
	conflictLogger := conflict_detector.NoopLogger
	if this.conflictDetector != nil {
		conflictLogger = this.conflictDetector.NewLogger(author)
	}
	return taraxa_state_db.New(db, conflictLogger)
}

func (this *stateTransition) newStateDB() *state.StateDB {
	stateDB, err := state.New(this.BaseStateRoot, this.StateDB)
	this.err.CheckIn(err)
	return stateDB
}

func (this *stateTransition) commitTransaction(db StateDB) (stateRootBytes []byte) {
	blockNumber := this.Block.Number
	chainConfig := this.Genesis.Config
	if chainConfig.IsByzantium(blockNumber) {
		db.Finalise(true)
	} else {
		stateRootBytes = db.IntermediateRoot(chainConfig.IsEIP158(blockNumber)).Bytes()
	}
	return
}
