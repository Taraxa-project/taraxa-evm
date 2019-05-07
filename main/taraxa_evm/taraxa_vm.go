package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_state_db"
	"math/big"
)

var sequential_group conflict_detector.Author = "SEQUENTIAL_GROUP"

type TaraxaVM struct {
	StaticConfig
	ExternalApi  api.ExternalApi
	ReadStateDB  state.Database
	WriteStateDB state.Database
}

func (this *TaraxaVM) GenerateSchedule(
	stateTransition *api.StateTransitionRequest) (
	result *api.ConcurrentSchedule, err error,
) {
	//var errFatal util.ErrorBarrier
	//defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	//txCount := len(req.Transactions)
	//conflictDetector := conflict_detector.New((txCount + 1) * 60)
	//go conflictDetector.Run()
	//defer conflictDetector.RequestShutdown()
	//parallelRoundDone := barrier.New(txCount)
	//stateRoot := req.StateRoot
	//for txId := 0; txId < txCount; txId++ {
	//	txId := txId
	//	go func() {
	//		var errConflict util.ErrorBarrier
	//		defer util.Recover(
	//			errFatal.Catch(),
	//			errConflict.Catch(),
	//		)
	//		defer parallelRoundDone.CheckIn()
	//		diskDb := this.ReadStateDB.TrieDB().GetDiskDB()
	//		diskDb.Get(stateRoot.Bytes())
	//		ethereumStateDB, stateDBCreateErr := state.New(stateRoot, this.ReadStateDB)
	//		errFatal.CheckIn(stateDBCreateErr)
	//		taraxaDB := taraxa_state_db.New(ethereumStateDB, conflictDetector.NewLogger(txId))
	//		result := this.executeTransaction(txId, req, taraxaDB, func(pc uint64) (uint64, bool) {
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

func (this *TaraxaVM) TransitionStateLikeEthereum(
	req *api.StateTransitionRequest, schedule *api.ConcurrentSchedule) (
	ret *api.StateTransitionResult, err error,
) {
	return stateTransition{this, req, schedule}.RunLikeEthereum()
}

type transactionResult struct {
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

func (this *TaraxaVM) executeTransaction(
	txId api.TxId,
	stateTransition *api.StateTransitionRequest,
	stateDB StateDB,
	controller vm.ExecutionController,
	gasPool *core.GasPool,
) *transactionResult {
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
	result := new(transactionResult)
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
