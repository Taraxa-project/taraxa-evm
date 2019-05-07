package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type stateTransition struct {
	*TaraxaVM
	req      *api.StateTransitionRequest
	schedule *api.ConcurrentSchedule
}

func (this stateTransition) Run() (ret *api.StateTransitionResult, err error) {
	ret = new(api.StateTransitionResult)
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	//commitDb := this.ReadStateDB
	//commitDb := this.WriteStateDB
	//if commitDb == nil {
	//	panic("don't go here")
	//	commitDb = state.NewDatabase(this.ReadStateDB.TrieDB().GetDiskDB())
	//}
	block := this.req.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		diskDB := this.ReadStateDB.TrieDB().GetDiskDB()
		_, _, genesisSetupErr := core.SetupGenesisBlock(diskDB, this.Genesis)
		errFatal.CheckIn(genesisSetupErr)
		ret.StateRoot = this.Genesis.ToBlock(diskDB).Root()
		return
	}
	transactions := block.Transactions
	txCount := len(transactions)
	interpreterController := vm.NoopExecutionController
	//sequentialTxCount := len(schedule.Sequential)
	//opLoggerFactory := conflict_detector.NoopLoggerFactory
	//if txCount != sequentialTxCount {
	//	panic("no")
	//	conflictDetector := conflict_detector.New((txCount + 1) * 60)
	//	tryReportConflicts := func() {
	//		conflictingAuthors := conflictDetector.RequestShutdown().Reset()
	//		if !conflictingAuthors.Empty() {
	//			errFatal.CheckIn(errors.New("Conflicts detected: " + conflictingAuthors.String()))
	//		}
	//	}
	//	opLoggerFactory = conflictDetector.NewLogger
	//	interpreterController = func(pc uint64) (uint64, bool) {
	//		errFatal.CheckIn()
	//		if conflictDetector.HaveBeenConflicts() {
	//			tryReportConflicts()
	//		}
	//		return pc, true
	//	}
	//	go conflictDetector.Run()
	//	defer tryReportConflicts()
	//}
	currentRoot := this.req.StateRoot
	//parallelResults := make(chan *transactionResult, txCount-sequentialTxCount)
	//var parallelCommitLock sync.Mutex
	//for txId, currentSeqIndex := 0, 0; txId < txCount; txId++ {
	//	if currentSeqIndex < sequentialTxCount && txId == schedule.Sequential[currentSeqIndex] {
	//		currentSeqIndex++
	//		continue
	//	}
	//	panic("don't go here")
	//	txId := txId
	//	go func() {
	//		defer util.Recover(errFatal.Catch(func(error) {
	//			util.Try(func() {
	//				close(parallelResults)
	//			})
	//		}))
	//		ethereumStateDB, stateDBCreateErr := state.New(req.StateRoot, this.ReadStateDB)
	//		errFatal.CheckIn(stateDBCreateErr)
	//		taraxaDb := taraxa_state_db.New(ethereumStateDB, opLoggerFactory(txId))
	//		result := this.executeTransaction(txId, req, taraxaDb, interpreterController)
	//		errFatal.CheckIn(result.dbErr, result.consensusErr)
	//		defer util.Try(func() {
	//			parallelResults <- result
	//		})
	//		parallelCommitLock.Lock()
	//		defer parallelCommitLock.Unlock()
	//		rebaseErr := ethereumStateDB.Rebase(currentRoot, commitDb)
	//		errFatal.CheckIn(rebaseErr)
	//		root, commitErr := ethereumStateDB.Commit(chainConfig.IsEIP158(blockNumber))
	//		errFatal.CheckIn(commitErr)
	//		currentRoot = root
	//	}()
	//}
	txResults := make([]*transactionResult, txCount)
	//for i := 0; i < cap(parallelResults); i++ {
	//	result := <-parallelResults
	//	errFatal.CheckIn()
	//	txResults[result.txId] = result
	//}
	finalStateDB, stateDbCreateErr := state.New(currentRoot, this.ReadStateDB)
	errFatal.CheckIn(stateDbCreateErr)
	//taraxaDB := taraxa_state_db.New(finalStateDB, opLoggerFactory(sequential_group))
	//gasPool := new(core.GasPool).AddGas(^uint64(0))
	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	chainConfig := this.Genesis.Config
	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(blockNumber) == 0 {
		misc.ApplyDAOHardFork(finalStateDB)
	}
	for _, txId := range this.schedule.Sequential {
		//fmt.Println("executing transaction", txId)
		result := this.executeTransaction(txId, this.req, finalStateDB, interpreterController, gasPool)
		//fmt.Println("gas", txId, gasPool.String(), req.Transactions[txId].GasLimit, result.gasUsed)
		//util.PanicIfPresent(result.contractErr)
		errFatal.CheckIn(result.consensusErr)
		//if result.contractErr != nil {
		//	fmt.Println("contract err", result.contractErr.Error())
		//	fmt.Println("gas used", result.gasUsed)
		//}
		//errFatal.CheckIn(result.dbErr)
		txResults[txId] = result
	}
	//gasPool := new(core.GasPool).AddGas(req.Block.GasLimit)
	for txId, txResult := range txResults {
		txData := transactions[txId]
		//nonceErr := core.CheckNonce(finalStateDB, txData.From, txData.Nonce)
		//errFatal.CheckIn(nonceErr)
		//for account, nonce := range txResult.transientState.NonceDeltas {
		//	nonceStr := strconv.FormatUint(nonce, 10)
		//	if !finalStateDB.Exist(account) { // TODO eliminate the need for this check
		//		panic(fmt.Sprintf("skipping nonce %s == %s\n", account.Hex(), nonceStr))
		//		continue
		//	}
		//	finalStateDB.AddNonce(account, nonce)
		//}
		// TODO turn on balances
		//for account, balanceDelta := range txResult.transientState.BalanceDeltas {
		//if !finalStateDB.Exist(account) || balanceDelta.Sign() == 0 {
		//	continue
		//}
		//finalStateDB.AddBalance(account, balanceDelta)
		//if finalStateDB.GetBalance(account).Sign() < 0 {
		//	// TODO record and replay validation events
		//	//errFatal.CheckIn(vm.ErrInsufficientBalance)
		//}
		//}
		//fmt.Println("gas", txId, gasPool.String(), txData.GasLimit, txResult.gasUsed)
		//gasLimitReachedErr := gasPool.SubGas(txData.GasLimit)
		//errFatal.CheckIn(gasLimitReachedErr)
		//gasPool.AddGas(txData.GasLimit - txResult.gasUsed)
		ret.UsedGas += txResult.gasUsed
		ethReceipt := types.NewReceipt(txResult.rootBytes, txResult.contractErr != nil, ret.UsedGas)
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
	ethash.AccumulateRewards(chainConfig, finalStateDB, block.HeaderNumerAndCoinbase, block.Uncles...)
	finalRoot, commitErr := finalStateDB.Commit(chainConfig.IsEIP158(blockNumber))
	errFatal.CheckIn(commitErr)
	util.Assert(finalRoot == this.req.ExpectedRoot)
	trieDB := this.ReadStateDB.TrieDB()
	//trieDB.SetDiskDB(commitTo)
	finalCommitErr := trieDB.Commit(finalRoot, false)
	errFatal.CheckIn(finalCommitErr)
	ret.StateRoot = finalRoot
	return
}

func (this stateTransition) RunLikeEthereum() (ret *api.StateTransitionResult, err error) {
	ret = new(api.StateTransitionResult)
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	block := this.req.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		diskDB := this.ReadStateDB.TrieDB().GetDiskDB()
		_, _, genesisSetupErr := core.SetupGenesisBlock(diskDB, this.Genesis)
		errFatal.CheckIn(genesisSetupErr)
		ret.StateRoot = this.Genesis.ToBlock(nil).Root()
		return
	}
	finalStateDB, stateDbCreateErr := state.New(this.req.StateRoot, this.ReadStateDB)
	errFatal.CheckIn(stateDbCreateErr)
	chainConfig := this.Genesis.Config
	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(blockNumber) == 0 {
		misc.ApplyDAOHardFork(finalStateDB)
	}
	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	for txId := range block.Transactions {
		txResult := this.executeTransaction(txId, this.req, finalStateDB, vm.NoopExecutionController, gasPool)
		errFatal.CheckIn(txResult.consensusErr)
		txData := block.Transactions[txId]
		ret.UsedGas += txResult.gasUsed
		ethReceipt := types.NewReceipt(txResult.rootBytes, txResult.contractErr != nil, ret.UsedGas)
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
	ethash.AccumulateRewards(chainConfig, finalStateDB, block.HeaderNumerAndCoinbase, block.Uncles...)
	finalRoot, commitErr := finalStateDB.Commit(chainConfig.IsEIP158(blockNumber))
	errFatal.CheckIn(commitErr)
	util.Assert(finalRoot == this.req.ExpectedRoot)
	finalCommitErr := this.ReadStateDB.TrieDB().Commit(finalRoot, false)
	errFatal.CheckIn(finalCommitErr)
	ret.StateRoot = finalRoot
	return
}
