package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"math/big"
)

var sequential_group conflict_detector.Author = "SEQUENTIAL_GROUP"

type TaraxaVM struct {
	StaticConfig
	ExternalApi api.ExternalApi
	StateDB     state.Database
	WriteDB     ethdb.Database
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
	//stateRoot := req.BaseStateRoot
	//for txId := 0; txId < txCount; txId++ {
	//	txId := txId
	//	go func() {
	//		var errConflict util.ErrorBarrier
	//		defer util.Recover(
	//			errFatal.Catch(),
	//			errConflict.Catch(),
	//		)
	//		defer parallelRoundDone.CheckIn()
	//		diskDb := this.StateDB.TrieDB().GetDiskDB()
	//		diskDb.Get(stateRoot.Bytes())
	//		ethereumStateDB, stateDBCreateErr := state.New(stateRoot, this.StateDB)
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
	//result.SequentialTransactions = make([]api.TxId, conflictingAuthors.Size())
	//conflictingAuthors.Each(func(index int, author conflict_detector.Author) {
	//	result.SequentialTransactions[index] = author.(api.TxId)
	//})
	//sort.Ints(result.SequentialTransactions)
	return
}

func (this *TaraxaVM) TransitionStateLikeEthereum(
	req *api.StateTransitionRequest, schedule *api.ConcurrentSchedule) (
	*api.StateTransitionResult, Metrics, error,
) {
	return stateTransition{
		TaraxaVM:               this,
		StateTransitionRequest: req,
		ConcurrentSchedule:     schedule,
	}.RunLikeEthereum()
}

type transactionRequest struct {
	txId                  api.TxId
	tx                    *api.Transaction
	blockHeader           *api.BlockHeader
	stateDB               StateDB
	interpreterController vm.ExecutionController
	gasPool               *core.GasPool
	checkNonce            bool
}

type transactionResult struct {
	vmReturnValue []byte
	gasUsed       uint64
	vmErr         error
	consensusErr  error
}

func (this *TaraxaVM) executeTransaction(r *transactionRequest) *transactionResult {
	block, tx := r.blockHeader, r.tx
	chainConfig := this.Genesis.Config
	blockNumber := block.Number
	evmContext := vm.Context{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     this.ExternalApi.GetHeaderHashByBlockNumber,
		Origin:      tx.From,
		Coinbase:    block.Coinbase,
		BlockNumber: blockNumber,
		Time:        block.Time,
		Difficulty:  block.Difficulty,
		GasLimit:    block.GasLimit,
		GasPrice:    new(big.Int).Set(tx.GasPrice),
	}
	evmConfig := vm.Config{
		StaticConfig: *this.EvmConfig,
	}
	evm := vm.NewEVMWithInterpreter(
		evmContext, r.stateDB, chainConfig, evmConfig,
		func(evm *vm.EVM) vm.Interpreter {
			return vm.NewEVMInterpreterWithExecutionController(evm, evmConfig, r.interpreterController)
		},
	)
	st := core.NewStateTransition(evm, tx.AsMessage(r.checkNonce), r.gasPool)
	r.stateDB.OpenTransaction(tx.Hash, block.Hash, r.txId)
	ret, usedGas, vmErr, consensusErr := st.TransitionDb()
	return &transactionResult{ret, usedGas, vmErr, consensusErr}
}

func (this *TaraxaVM) PersistentCommit(root common.Hash) error {
	trieDb := this.StateDB.TrieDB()
	originalDiskDb := trieDb.GetDiskDB()
	defer trieDb.SetDiskDB(originalDiskDb)
	trieDb.SetDiskDB(this.WriteDB)
	return trieDb.Commit(root, false)
}
