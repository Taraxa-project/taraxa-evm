package taraxa_evm

import (
	"errors"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_state_db"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/main/util/barrier"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
	"sort"
	"strconv"
	"sync"
)

var sequential_group conflict_detector.Author = "SEQUENTIAL_GROUP"

type TaraxaEvm struct {
	ExternalApi   api.ExternalApi
	ChainConfig   *params.ChainConfig
	EvmConfig     *vm.StaticConfig
	StateDatabase state.Database
}

func (this *TaraxaEvm) GenerateSchedule(
	stateTransition *api.StateTransition,
) (result *api.ConcurrentSchedule, err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	txCount := len(stateTransition.Transactions)
	conflictDetector := conflict_detector.New((txCount + 1) * 60)
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	parallelRoundDone := barrier.New(txCount)
	stateRoot := stateTransition.StateRoot
	for txId := 0; txId < txCount; txId++ {
		txId := txId
		go func() {
			var errConflict util.ErrorBarrier
			defer util.Recover(
				errFatal.Catch(),
				errConflict.Catch(),
			)
			defer parallelRoundDone.CheckIn()
			diskDb := this.StateDatabase.TrieDB().GetDiskDB()
			diskDb.Get(stateRoot.Bytes())
			ethereumStateDB, stateDBCreateErr := state.New(stateRoot, this.StateDatabase)
			errFatal.CheckIn(stateDBCreateErr)
			taraxaDB := taraxa_state_db.New(ethereumStateDB, conflictDetector.NewLogger(txId))
			result := this.ExecuteTransaction(txId, stateTransition, taraxaDB, func(pc uint64) (uint64, bool) {
				errFatal.CheckIn()
				if conflictDetector.IsCurrentlyInConflict(txId) {
					errConflict.CheckIn(errors.New("CONFLICT"))
				}
				return pc, true
			})
			errFatal.CheckIn(result.dbErr, result.consensusErr)
		}()
	}
	parallelRoundDone.Await()
	errFatal.CheckIn()
	conflictingAuthors := conflictDetector.RequestShutdown().Reset()
	result = new(api.ConcurrentSchedule)
	result.Sequential = make([]api.TxId, conflictingAuthors.Size())
	conflictingAuthors.Each(func(index int, author conflict_detector.Author) {
		result.Sequential[index] = author.(api.TxId)
	})
	sort.Ints(result.Sequential)
	return
}

func (this *TaraxaEvm) TransitionState(
	stateTransition *api.StateTransition,
	schedule *api.ConcurrentSchedule,
	commitTo ethdb.Database,
) (ret *api.StateTransitionResult, err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	diskDB := this.StateDatabase.TrieDB().GetDiskDB()
	if commitTo == nil {
		commitTo = diskDB
	}
	commitDb := state.NewDatabase(diskDB)
	txCount := len(stateTransition.Transactions)
	sequentialTxCount := len(schedule.Sequential)
	conflictDetector := conflict_detector.New((txCount + 1) * 60)
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	interpreterAborter := func(pc uint64) (uint64, bool) {
		errFatal.CheckIn()
		if conflictDetector.HaveBeenConflicts() {
			errFatal.CheckIn(errors.New("CONFLICT"))
		}
		return pc, true
	}
	currentRoot := stateTransition.StateRoot
	parallelResults := make(chan *TransactionResult, txCount-sequentialTxCount)
	var parallelCommitLock sync.Mutex
	blockNumber := api.BigInt(stateTransition.Block.Number)
	for txId, currentSeqIndex := 0, 0; txId < txCount; txId++ {
		if currentSeqIndex < sequentialTxCount && txId == schedule.Sequential[currentSeqIndex] {
			currentSeqIndex++
			continue
		}
		txId := txId
		go func() {
			defer util.Recover(errFatal.Catch(func(error) {
				util.Try(func() {
					close(parallelResults)
				})
			}))
			ethereumStateDB, stateDBCreateErr := state.New(stateTransition.StateRoot, this.StateDatabase)
			errFatal.CheckIn(stateDBCreateErr)
			taraxaDb := taraxa_state_db.New(ethereumStateDB, conflictDetector.NewLogger(txId))
			result := this.ExecuteTransaction(txId, stateTransition, taraxaDb, interpreterAborter)
			errFatal.CheckIn(result.dbErr, result.consensusErr)
			defer util.Try(func() {
				parallelResults <- result
			})
			parallelCommitLock.Lock()
			defer parallelCommitLock.Unlock()
			rebaseErr := ethereumStateDB.Rebase(currentRoot, commitDb)
			errFatal.CheckIn(rebaseErr)
			root, commitErr := ethereumStateDB.Commit(this.ChainConfig.IsEIP158(blockNumber))
			b := ethereumStateDB.Exist(common.HexToAddress("0xcb350b1D62684c80Cf15696c28550B343A0c6444"))
			util.Noop(b)
			errFatal.CheckIn(commitErr)
			currentRoot = root
		}()
	}
	txResults := make([]*TransactionResult, txCount)
	for i := 0; i < cap(parallelResults); i++ {
		result := <-parallelResults
		errFatal.CheckIn()
		txResults[result.txId] = result
	}
	finalStateDB, stateDbCreateErr := state.New(currentRoot, commitDb)
	errFatal.CheckIn(stateDbCreateErr)
	taraxaDB := taraxa_state_db.New(finalStateDB, conflictDetector.NewLogger(sequential_group))
	for _, txId := range schedule.Sequential {
		currentRootStr := currentRoot.Hex()
		util.Noop(currentRootStr)
		result := this.ExecuteTransaction(txId, stateTransition, taraxaDB, interpreterAborter)
		errFatal.CheckIn(result.consensusErr, result.dbErr)
		txResults[txId] = result
	}
	conflictDetector.RequestShutdown()
	gasPool := new(core.GasPool).AddGas(stateTransition.Block.GasLimit)
	ret = new(api.StateTransitionResult)
	for txId, txResult := range txResults {
		txData := stateTransition.Transactions[txId]
		nonceErr := core.CheckNonce(finalStateDB, txData.From, txData.Nonce)
		errFatal.CheckIn(nonceErr)
		for account, nonce := range txResult.transientState.NonceDeltas {
			if !finalStateDB.Exist(account) { // TODO eliminate the need for this check
				panic(fmt.Sprintf("skipping nonce %s == %s\n", account.Hex(), strconv.FormatUint(nonce, 10)))
				continue
			}
			finalStateDB.AddNonce(account, nonce)
		}
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
	}
	finalRoot, commitErr := finalStateDB.Commit(this.ChainConfig.IsEIP158(blockNumber))
	errFatal.CheckIn(commitErr)
	trieDB := commitDb.TrieDB()
	trieDB.SetDiskDB(commitTo)
	finalCommitErr := trieDB.Commit(finalRoot, false)
	errFatal.CheckIn(finalCommitErr)
	conflictingAuthors := conflictDetector.Reset()
	if !conflictingAuthors.Empty() {
		errFatal.CheckIn(errors.New("Conflicts detected: " + conflictingAuthors.String()))
	}
	ret.StateRoot = finalRoot
	return
}

type TransactionResult struct {
	txId           api.TxId
	value          hexutil.Bytes
	gasUsed        uint64
	logs           []*types.Log
	transientState *taraxa_state_db.TransientState
	contractErr    error
	consensusErr   error
	dbErr          error
}

func (this *TaraxaEvm) ExecuteTransaction(
	txId api.TxId,
	stateTransition *api.StateTransition,
	taraxaDB *taraxa_state_db.TaraxaStateDB,
	controller vm.ExecutionController,
) *TransactionResult {
	block := stateTransition.Block
	txData := stateTransition.Transactions[txId]
	gasPrice := api.BigInt(txData.GasPrice)
	evmContext := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     this.ExternalApi.GetHeaderHashByBlockNumber,
		Origin:      txData.From,
		Coinbase:    block.Coinbase,
		BlockNumber: api.BigInt(block.Number),
		Time:        api.BigInt(block.Time),
		Difficulty:  api.BigInt(block.Difficulty),
		GasLimit:    block.GasLimit,
		GasPrice:    new(big.Int).Set(gasPrice),
	}
	evmConfig := vm.Config{
		StaticConfig: *this.EvmConfig,
	}
	evm := vm.NewEVMWithInterpreter(
		evmContext, taraxaDB, this.ChainConfig, evmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, controller)
		},
	)
	message := types.NewMessage(
		txData.From, txData.To, txData.Nonce, api.BigInt(txData.Amount), txData.GasLimit, gasPrice, *txData.Data,
		false,
	)
	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	st := core.NewStateTransition(evm, message, gasPool)
	taraxaDB.Prepare(txData.Hash, block.Hash, txId)
	result := new(TransactionResult)
	result.value, result.gasUsed, result.contractErr, result.consensusErr = st.TransitionDb()
	result.txId = txId
	result.dbErr = taraxaDB.Error()
	result.logs = taraxaDB.GetLogs(txData.Hash)
	result.transientState = taraxaDB.ResetCurrentTransientState()
	return result
}
