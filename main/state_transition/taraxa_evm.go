package state_transition

import (
	"errors"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/external_api"
	"github.com/Taraxa-project/taraxa-evm/main/state_db"
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
	ExternalApi     *external_api.ExternalApi
	ChainConfig     *params.ChainConfig
	EvmConfig       *vm.Config
	StateTransition *api.StateTransition
	StateDatabase   state.Database
}

func (this *TaraxaEvm) GenerateSchedule() (result *api.ConcurrentSchedule, err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	txCount := len(this.StateTransition.Transactions)
	conflictDetector := new(conflict_detector.ConflictDetector).Init(txCount * 60)
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	parallelRoundDone := barrier.New(txCount)
	for txId := 0; txId < txCount; txId++ {
		txId := txId
		go func() {
			var errConflict util.ErrorBarrier
			defer util.Recover(
				errFatal.Catch(),
				errConflict.Catch(),
			)
			defer parallelRoundDone.CheckIn()
			ethereumStateDB, stateDBCreateErr := state.New(this.StateTransition.StateRoot, this.StateDatabase)
			errFatal.CheckIn(stateDBCreateErr)
			taraxaDB := state_db.NewDB(ethereumStateDB, conflictDetector.NewLogger(txId))
			result := this.Run(txId, taraxaDB, func(pc uint64) (uint64, bool) {
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
	fmt.Println("passed the barrier")
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

func (this *TaraxaEvm) TransitionState(schedule *api.ConcurrentSchedule) (ret *api.StateTransitionResult, err error) {
	ret = new(api.StateTransitionResult)
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(func(e error) {
		err = e
	}))
	txCount := len(this.StateTransition.Transactions)
	sequentialTxCount := len(schedule.Sequential)
	conflictDetector := new(conflict_detector.ConflictDetector).Init((txCount + 1) * 6000)
	go conflictDetector.Run()
	defer conflictDetector.RequestShutdown()
	interpreterAborter := func(pc uint64) (uint64, bool) {
		errFatal.CheckIn()
		if conflictDetector.HaveBeenConflicts() {
			errFatal.CheckIn(errors.New("CONFLICT"))
		}
		return pc, true
	}
	diskDb := this.StateDatabase.TrieDB().DiskDB().(ethdb.Database)
	var parallelCommitLock sync.Mutex
	commitDb := state.NewDatabase(diskDb)
	currentRoot := this.StateTransition.StateRoot
	parallelResults := make(chan *TransactionResult, txCount-sequentialTxCount)
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
			var result *TransactionResult
			defer util.Try(func() {
				parallelResults <- result
			})
			ethereumStateDB, stateDBCreateErr := state.New(this.StateTransition.StateRoot, this.StateDatabase)
			errFatal.CheckIn(stateDBCreateErr)
			taraxaDb := state_db.NewDB(ethereumStateDB, conflictDetector.NewLogger(txId))
			result = this.Run(txId, taraxaDb, interpreterAborter)
			errFatal.CheckIn(result.dbErr, result.consensusErr)
			parallelCommitLock.Lock()
			defer parallelCommitLock.Unlock()
			rebaseErr := ethereumStateDB.Rebase(currentRoot, commitDb)
			errFatal.CheckIn(rebaseErr)
			root, commitErr := ethereumStateDB.Commit(true)
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
	taraxaDB := state_db.NewDB(finalStateDB, conflictDetector.NewLogger(sequential_group))
	for _, txId := range schedule.Sequential {
		result := this.Run(txId, taraxaDB, interpreterAborter)
		errFatal.CheckIn(result.consensusErr, result.dbErr)
		txResults[txId] = result
	}
	conflictDetector.RequestShutdown()
	gasPool := new(core.GasPool).AddGas(this.StateTransition.Block.GasLimit)
	for txId, txResult := range txResults {
		txData := this.StateTransition.Transactions[txId]
		nonceErr := core.CheckNonce(finalStateDB, txData.From, txData.Nonce)
		errFatal.CheckIn(nonceErr)
		for account, nonce := range txResult.transientState.NonceDeltas {
			if !finalStateDB.Exist(account) { // TODO eliminate the need for this check
				panic(fmt.Sprintf("skipping nonce %s == %s\n", account.Hex(), strconv.FormatUint(nonce, 10)))
				continue
			}
			finalStateDB.AddNonce(account, nonce)
		}
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
	finalRoot, commitErr := finalStateDB.Commit(true)
	errFatal.CheckIn(commitErr)
	finalCommitErr := commitDb.TrieDB().Commit(finalRoot, true)
	errFatal.CheckIn(finalCommitErr)
	conflictingAuthors := conflictDetector.Reset()
	if !conflictingAuthors.Empty() {
		errFatal.CheckIn(errors.New("CONFLICTS: " + conflictingAuthors.String()))
	}
	ret.StateRoot = finalRoot
	return
}

type TransactionResult struct {
	txId           api.TxId
	value          hexutil.Bytes
	gasUsed        uint64
	logs           []*types.Log
	transientState *state_db.TransientState
	contractErr    error
	consensusErr   error
	dbErr          error
}

func (this *TaraxaEvm) Run(txId api.TxId, taraxaDB *state_db.TaraxaStateDB, controller vm.ExecutionController) *TransactionResult {
	block := this.StateTransition.Block
	txData := this.StateTransition.Transactions[txId]
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
	evm := vm.NewEVMWithInterpreter(
		evmContext, taraxaDB, this.ChainConfig, *this.EvmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, *this.EvmConfig, controller)
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
