package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_state_db"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

var sequential_group conflict_detector.Author = "SEQUENTIAL_GROUP"

type TaraxaVM struct {
	ExternalApi   api.ExternalApi
	Genesis       *core.Genesis
	EvmConfig     *vm.StaticConfig
	SourceStateDB state.Database
	TargetStateDB state.Database
}

func (this *TaraxaVM) GenerateSchedule(
	stateTransition *api.StateTransition) (
	result *api.ConcurrentSchedule, err error,
) {
	//var errFatal util.ErrorBarrier
	//defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	//txCount := len(stateTransition.Transactions)
	//conflictDetector := conflict_detector.New((txCount + 1) * 60)
	//go conflictDetector.Run()
	//defer conflictDetector.RequestShutdown()
	//parallelRoundDone := barrier.New(txCount)
	//stateRoot := stateTransition.StateRoot
	//for txId := 0; txId < txCount; txId++ {
	//	txId := txId
	//	go func() {
	//		var errConflict util.ErrorBarrier
	//		defer util.Recover(
	//			errFatal.Catch(),
	//			errConflict.Catch(),
	//		)
	//		defer parallelRoundDone.CheckIn()
	//		diskDb := this.SourceStateDB.TrieDB().GetDiskDB()
	//		diskDb.Get(stateRoot.Bytes())
	//		ethereumStateDB, stateDBCreateErr := state.New(stateRoot, this.SourceStateDB)
	//		errFatal.CheckIn(stateDBCreateErr)
	//		taraxaDB := taraxa_state_db.New(ethereumStateDB, conflictDetector.NewLogger(txId))
	//		result := this.ExecuteTransaction(txId, stateTransition, taraxaDB, func(pc uint64) (uint64, bool) {
	//			errFatal.CheckIn()
	//			if conflictDetector.IsCurrentlyInConflict(txId) {
	//				errConflict.CheckIn(errors.New("CONFLICT"))
	//			}
	//			return pc, true
	//		})
	//		errFatal.CheckIn(result.dbErr, result.consensusErr)
	//	}()
	//}
	//parallelRoundDone.Await()
	//errFatal.CheckIn()
	//conflictingAuthors := conflictDetector.RequestShutdown().Reset()
	//result = new(api.ConcurrentSchedule)
	//result.Sequential = make([]api.TxId, conflictingAuthors.Size())
	//conflictingAuthors.Each(func(index int, author conflict_detector.Author) {
	//	result.Sequential[index] = author.(api.TxId)
	//})
	//sort.Ints(result.Sequential)
	return
}

// TODO auto create block beneficiary accounts
func (this *TaraxaVM) TransitionState(
	stateTransition *api.StateTransition, schedule *api.ConcurrentSchedule) (
	ret *api.StateTransitionResult, err error,
) {
	ret = new(api.StateTransitionResult)
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	//commitDb := this.SourceStateDB
	//commitDb := this.TargetStateDB
	//if commitDb == nil {
	//	panic("don't go here")
	//	commitDb = state.NewDatabase(this.SourceStateDB.TrieDB().GetDiskDB())
	//}
	block := stateTransition.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		diskDB := this.SourceStateDB.TrieDB().GetDiskDB()
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
	currentRoot := stateTransition.StateRoot
	//parallelResults := make(chan *TransactionResult, txCount-sequentialTxCount)
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
	//		ethereumStateDB, stateDBCreateErr := state.New(stateTransition.StateRoot, this.SourceStateDB)
	//		errFatal.CheckIn(stateDBCreateErr)
	//		taraxaDb := taraxa_state_db.New(ethereumStateDB, opLoggerFactory(txId))
	//		result := this.ExecuteTransaction(txId, stateTransition, taraxaDb, interpreterController)
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
	txResults := make([]*TransactionResult, txCount)
	//for i := 0; i < cap(parallelResults); i++ {
	//	result := <-parallelResults
	//	errFatal.CheckIn()
	//	txResults[result.txId] = result
	//}
	finalStateDB, stateDbCreateErr := state.New(currentRoot, this.SourceStateDB)
	errFatal.CheckIn(stateDbCreateErr)
	//taraxaDB := taraxa_state_db.New(finalStateDB, opLoggerFactory(sequential_group))
	//gasPool := new(core.GasPool).AddGas(^uint64(0))
	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	chainConfig := this.Genesis.Config
	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(blockNumber) == 0 {
		misc.ApplyDAOHardFork(finalStateDB)
	}
	for _, txId := range schedule.Sequential {
		//fmt.Println("executing transaction", txId)
		result := this.ExecuteTransaction(txId, stateTransition, finalStateDB, interpreterController, gasPool)
		//fmt.Println("gas", txId, gasPool.String(), stateTransition.Transactions[txId].GasLimit, result.gasUsed)
		//util.PanicIfPresent(result.contractErr)
		errFatal.CheckIn(result.consensusErr)
		errFatal.CheckIn(result.dbErr)
		txResults[txId] = result
	}
	//gasPool := new(core.GasPool).AddGas(stateTransition.Block.GasLimit)
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
	trieDB := this.SourceStateDB.TrieDB()
	//trieDB.SetDiskDB(commitTo)
	finalCommitErr := trieDB.Commit(finalRoot, false)
	errFatal.CheckIn(finalCommitErr)
	ret.StateRoot = finalRoot
	return
}

type TransactionResult struct {
	txId           api.TxId
	value          hexutil.Bytes
	gasUsed        uint64
	logs           []*types.Log
	rootBytes      []byte
	transientState *taraxa_state_db.TransientState
	contractErr    error
	consensusErr   error
	dbErr          error
}

func (this *TaraxaVM) ExecuteTransaction(
	txId api.TxId,
	stateTransition *api.StateTransition,
//stateDB *taraxa_state_db.TaraxaStateDB,
	stateDB *state.StateDB,
	controller vm.ExecutionController,
	gasPool *core.GasPool,
) *TransactionResult {
	chainConfig := this.Genesis.Config
	block := stateTransition.Block
	txData := block.Transactions[txId]
	gasPrice := txData.GasPrice
	blockNumber := block.Number
	evmContext := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     this.ExternalApi.GetHeaderHashByBlockNumber,
		Origin:      txData.From,
		Coinbase:    block.Coinbase,
		BlockNumber: blockNumber,
		Time:        block.Time,
		Difficulty:  block.Difficulty,
		GasLimit:    block.GasLimit,
		GasPrice:    new(big.Int).Set(gasPrice),
	}
	evmConfig := vm.Config{
		StaticConfig: *this.EvmConfig,
	}
	evm := vm.NewEVMWithInterpreter(
		evmContext, stateDB, chainConfig, evmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, controller)
		},
	)
	message := types.NewMessage(
		txData.From, txData.To, txData.Nonce, txData.Amount, txData.GasLimit, gasPrice, *txData.Data,
		true,
	)
	//gasPool := new(core.GasPool).AddGas(block.GasLimit)
	st := core.NewStateTransition(evm, message, gasPool)
	stateDB.Prepare(txData.Hash, block.Hash, txId)
	result := new(TransactionResult)
	result.value, result.gasUsed, result.contractErr, result.consensusErr = st.TransitionDb()
	result.txId = txId
	result.dbErr = stateDB.Error()
	result.logs = stateDB.GetLogs(txData.Hash)
	if chainConfig.IsByzantium(blockNumber) {
		stateDB.Finalise(true)
	} else {
		result.rootBytes = stateDB.IntermediateRoot(chainConfig.IsEIP158(blockNumber)).Bytes()
	}
	//result.transientState = stateDB.CommitTransientState()
	return result
}
