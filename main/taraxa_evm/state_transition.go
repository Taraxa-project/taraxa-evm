package taraxa_evm

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

type stateTransition struct {
	*TaraxaVM
	*api.StateTransitionRequest
	*api.ConcurrentSchedule
	interpreterController vm.ExecutionController
	opLoggerFactory       conflict_detector.OperationLoggerFactory
	err                   util.ErrorBarrier
}

type parallelTxResult struct {
	txId     api.TxId
	txResult *transactionResult
	db       StateDB
}

func (this *stateTransition) init() {
	this.interpreterController = vm.NoopExecutionController
	this.opLoggerFactory = conflict_detector.NoopLoggerFactory
	transactions := this.Block.Transactions
	txCount := len(transactions)
	sequentialTxCount := this.SequentialTransactions.Size()
	if txCount == sequentialTxCount {
		return
	}
	panic("no")
	conflictDetector := conflict_detector.New((txCount + 1) * 60)
	tryReportConflicts := func() {
		conflictingAuthors := conflictDetector.RequestShutdown().Reset()
		if !conflictingAuthors.Empty() {
			this.err.CheckIn(errors.New("Conflicts detected: " + conflictingAuthors.String()))
		}
	}
	this.opLoggerFactory = conflictDetector.NewLogger
	this.interpreterController = func(pc uint64) (uint64, bool) {
		this.err.CheckIn()
		if conflictDetector.HaveBeenConflicts() {
			tryReportConflicts()
		}
		return pc, true
	}
	go conflictDetector.Run()
	defer tryReportConflicts()
}

//func (this stateTransition) Run() (ret *api.StateTransitionResult, err error) {
//	ret = new(api.StateTransitionResult)
//	defer util.Recover(this.err.Catch(util.SetTo(&err)))
//
//	block := this.Block
//	blockNumber := block.Number
//
//	if blockNumber.Sign() == 0 {
//		_, _, genesisSetupErr := core.SetupGenesisBlock(this.WriteDB, this.Genesis)
//		this.err.CheckIn(genesisSetupErr)
//		ret.BaseStateRoot = this.Genesis.ToBlock(nil).Root()
//		return
//	}
//
//	transactions := block.Transactions
//	txCount := len(transactions)
//	interpreterController := vm.NoopExecutionController
//	sequentialTxCount := this.SequentialTransactions.Size()
//	opLoggerFactory := conflict_detector.NoopLoggerFactory
//	if txCount != sequentialTxCount {
//		panic("no")
//		conflictDetector := conflict_detector.New((txCount + 1) * 60)
//		tryReportConflicts := func() {
//			conflictingAuthors := conflictDetector.RequestShutdown().Reset()
//			if !conflictingAuthors.Empty() {
//				this.err.CheckIn(errors.New("Conflicts detected: " + conflictingAuthors.String()))
//			}
//		}
//		opLoggerFactory = conflictDetector.NewLogger
//		interpreterController = func(pc uint64) (uint64, bool) {
//			this.err.CheckIn()
//			if conflictDetector.HaveBeenConflicts() {
//				tryReportConflicts()
//			}
//			return pc, true
//		}
//		go conflictDetector.Run()
//		defer tryReportConflicts()
//	}
//
//	parallelResults := make(chan *parallelTxResult, txCount-sequentialTxCount)
//	for txId := 0; txId < txCount; txId++ {
//		if this.SequentialTransactions.Contains(txId) {
//			continue
//		}
//		panic("don't go here")
//		txId := txId
//		go func() {
//			defer util.Recover(this.err.Catch(func(error) {
//				util.Try(func() {
//					close(parallelResults)
//				})
//			}))
//			ethereumStateDB, stateDBCreateErr := state.New(this.BaseStateRoot, this.StateDB)
//			this.err.CheckIn(stateDBCreateErr)
//			taraxaDb := taraxa_state_db.New(ethereumStateDB, opLoggerFactory(txId))
//			gasPool := new(core.GasPool).AddGas(math.MaxUint64)
//			result := this.executeTransaction(txId, block, taraxaDb, interpreterController, gasPool, true)
//			this.err.CheckIn(result.consensusErr)
//			util.Try(func() {
//				parallelResults <- &parallelTxResult{txId, result, ethereumStateDB}
//			})
//		}()
//	}
//
//	stateDB, stateDbCreateErr := state.New(this.BaseStateRoot, this.StateDB)
//	this.err.CheckIn(stateDbCreateErr)
//
//	chainConfig := this.Genesis.Config
//
//	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(blockNumber) == 0 {
//		misc.ApplyDAOHardFork(stateDB)
//	}
//
//	txResults := make([]*transactionResult, txCount)
//	for i := 0; i < cap(parallelResults); i++ {
//		result := <-parallelResults
//		this.err.CheckIn()
//		stateDB.MergeChanges(result.db)
//		txResults[result.txId] = result.txResult
//	}
//
//	taraxaDB := taraxa_state_db.New(stateDB, opLoggerFactory(sequential_group))
//	this.SequentialTransactions.Each(func(_ int, value interface{}) {
//		txId := value.(api.TxId)
//		gasPool := new(core.GasPool).AddGas(math.MaxUint64)
//		result := this.executeTransaction(txId, block, taraxaDB, interpreterController, gasPool, false)
//		this.err.CheckIn(result.consensusErr)
//		txResults[txId] = result
//	})
//
//	gasPool := new(core.GasPool).AddGas(block.GasLimit)
//	for txId, txResult := range txResults {
//		txData := transactions[txId]
//
//		nonceErr := core.CheckNonce(stateDB, txData.From, txData.Nonce)
//		this.err.CheckIn(nonceErr)
//
//		blockGasDepletedErr := gasPool.SubGas(txData.GasLimit)
//		this.err.CheckIn(blockGasDepletedErr)
//		gasPool.AddGas(txData.GasLimit - txResult.gasUsed)
//
//		for account, nonceDelta := range txResult.transientState.NonceDeltas {
//			if !stateDB.Exist(account) {
//				panic("account doesn't exist", account.Hex())
//			}
//			stateDB.AddNonce(account, nonceDelta)
//		}
//		for account, balanceDelta := range txResult.transientState.BalanceDeltas {
//			if !stateDB.Exist(account) {
//				panic("account doesn't exist", account.Hex())
//			}
//			//if balanceDelta.Sign() == 0 {
//			//	continue
//			//}
//			stateDB.AddBalance(account, balanceDelta)
//			if balanceDelta < 0 && stateDB.GetBalance(account).Sign() < 0 {
//				// TODO record and replay validation events
//				this.err.CheckIn(vm.ErrInsufficientBalance)
//			}
//		}
//
//		ret.UsedGas += txResult.gasUsed
//		ethReceipt := types.NewReceipt(txResult.rootBytes, txResult.contractErr != nil, ret.UsedGas)
//		if txData.To == nil {
//			ethReceipt.ContractAddress = crypto.CreateAddress(txData.From, txData.Nonce)
//		}
//		ethReceipt.TxHash = txData.Hash;
//		ethReceipt.GasUsed = txResult.gasUsed
//		ethReceipt.Logs = txResult.logs
//		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
//		ret.Receipts = append(ret.Receipts, &api.TaraxaReceipt{
//			ReturnValue:     txResult.value,
//			ContractError:   txResult.contractErr,
//			EthereumReceipt: ethReceipt,
//		})
//		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
//	}
//
//	ethash.AccumulateRewards(chainConfig, stateDB, block.HeaderNumerAndCoinbase, block.Uncles...)
//
//	finalRoot, commitErr := stateDB.Commit(chainConfig.IsEIP158(blockNumber))
//	ret.BaseStateRoot = finalRoot
//	this.err.CheckIn(commitErr)
//	util.Assert(finalRoot == this.ExpectedRoot)
//
//	trieDB := this.StateDB.TrieDB()
//	oldDiskDb := trieDB.GetDiskDB()
//	defer trieDB.SetDiskDB(oldDiskDb)
//	trieDB.SetDiskDB(this.WriteDB)
//	finalCommitErr := trieDB.Commit(finalRoot, false)
//	this.err.CheckIn(finalCommitErr)
//	return
//}

func (this stateTransition) RunLikeEthereum() (ret *api.StateTransitionResult, err error) {
	//TODO remove
	this.interpreterController = vm.NoopExecutionController
	defer util.Recover(this.err.Catch(util.SetTo(&err)))
	ret = new(api.StateTransitionResult)
	block := this.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		diskDB := this.StateDB.TrieDB().GetDiskDB()
		_, _, genesisSetupErr := core.SetupGenesisBlock(diskDB, this.Genesis)
		this.err.CheckIn(genesisSetupErr)
		ret.StateRoot = this.Genesis.ToBlock(nil).Root()
		return
	}
	stateDB, stateDbCreateErr := state.New(this.BaseStateRoot, this.StateDB)
	this.err.CheckIn(stateDbCreateErr)
	chainConfig := this.Genesis.Config
	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(blockNumber) == 0 {
		misc.ApplyDAOHardFork(stateDB)
	}
	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	for txId := range block.Transactions {
		taraxaReceipt := this.executeTransaction(txId, stateDB, gasPool, true)
		rootBytes := this.finalizeStateDB(stateDB, block.Number)
		ethReceipt := taraxaReceipt.EthereumReceipt
		ethReceipt.PostState = rootBytes
		ret.UsedGas += ethReceipt.GasUsed
		ethReceipt.CumulativeGasUsed = ret.UsedGas
		ret.Receipts = append(ret.Receipts, taraxaReceipt)
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}
	ethash.AccumulateRewards(chainConfig, stateDB, &block.HeaderNumerAndCoinbase, block.Uncles...)
	finalRoot, commitErr := stateDB.Commit(chainConfig.IsEIP158(blockNumber))
	this.err.CheckIn(commitErr)
	util.Assert(finalRoot == this.ExpectedRoot)
	finalCommitErr := this.StateDB.TrieDB().Commit(finalRoot, false)
	this.err.CheckIn(finalCommitErr)
	ret.StateRoot = finalRoot
	return
}

func (this *stateTransition) executeTransaction(txId api.TxId, db StateDB, gp *core.GasPool, checkNonce bool) *api.TaraxaReceipt {
	block := this.Block
	tx := block.Transactions[txId]
	result := this.TaraxaVM.executeTransaction(&transactionRequest{
		txId, tx, &block.BlockHeader, db, this.interpreterController, gp, checkNonce,
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

func (this *stateTransition) finalizeStateDB(db StateDB, blockNumber *big.Int) (stateRootBytes []byte) {
	chainConfig := this.Genesis.Config
	if chainConfig.IsByzantium(blockNumber) {
		db.Finalise(true)
	} else {
		stateRootBytes = db.IntermediateRoot(chainConfig.IsEIP158(blockNumber)).Bytes()
	}
	return
}
