package vm

import (
	"errors"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	eth_math "github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/taraxa/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
)

type StateTransitionMetrics struct {
	TransactionMetrics       []TransactionMetrics       `json:"transactionMetrics"`
	TotalTime                metric_utils.AtomicCounter `json:"totalTime"`
	TrieCommitSync           metric_utils.AtomicCounter `json:"trieCommitSync"`
	ConflictDetectionSync    metric_utils.AtomicCounter `json:"conflictDetectionSync"`
	PostProcessingSync       metric_utils.AtomicCounter `json:"postProcessingSync"`
	ParallelTransactionsSync metric_utils.AtomicCounter `json:"parallelTransactionsSync"`
	SequentialTransactions   metric_utils.AtomicCounter `json:"sequentialTransactions"`
	TrieCommitTotal          metric_utils.AtomicCounter `json:"trieCommitTotal"`
	PersistentCommit         metric_utils.AtomicCounter `json:"persistentCommit"`
}

type stateTransition struct {
	*VM
	*StateTransitionRequest
	*ConcurrentSchedule
	result  *StateTransitionResult
	metrics *StateTransitionMetrics
	err     util.ErrorBarrier
}

func newStateTransition(vm *VM, req *StateTransitionRequest, sched *ConcurrentSchedule) *stateTransition {
	return &stateTransition{
		VM:                     vm,
		StateTransitionRequest: req,
		ConcurrentSchedule:     sched,
		result:                 &StateTransitionResult{},
		metrics: &StateTransitionMetrics{
			TransactionMetrics: make([]TransactionMetrics, len(req.Block.Transactions)),
		},
	}
}

func (this *stateTransition) run() {
	defer util.Recover(this.err.Catch())
	block := this.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		this.result.StateRoot = this.applyGenesisBlock()
		return
	}
	txCount := len(block.Transactions)
	parallelTxCount := txCount - this.SequentialTransactions.Size()
	conflictDetector := conflict_detector.New(
		(parallelTxCount+1)*this.ConflictDetectorInboxPerTransaction,
		func(op *conflict_detector.Operation, authors conflict_detector.Authors) {
			this.err.SetIfAbsent(errors.New(fmt.Sprintf(
				"Conflict detected. Operation: {%s, %s, %s}; Authors: %s",
				op.Author, op.Type, op.Key, util.Join(", ", authors.Values())),
			))
		})
	go conflictDetector.Run()
	defer conflictDetector.Halt()
	committer := LaunchStateDBCommitter(txCount+1, this.newStateDBForReading, this.commitToTrieAndCommitTrie)
	defer committer.Halt()
	postProcessor := LaunchBlockPostProcessor(block, this.newStateDBForReading, func(err error) {
		this.err.SetIfAbsent(err)
	})
	defer postProcessor.Halt()
	defer this.metrics.TotalTime.Recorder()()
	parallelTxSyncMeter := this.metrics.ParallelTransactionsSync.Recorder()
	parallelStateChanges := make(chan state.StateChange, parallelTxCount)
	concurrent.Parallelize(this.NumConcurrentProcesses, txCount, func(int) func(int) {
		stateDB := this.newStateDBForReading()
		return func(txId TxId) {
			if this.SequentialTransactions.Contains(txId) {
				return
			}
			defer util.Recover(this.err.Catch(func(error) {
				concurrent.TryClose(parallelStateChanges)
			}))
			this.applyHardForks(stateDB)
			result := this.executeTransaction(txId, &OperationLoggingStateDB{
				stateDB,
				conflictDetector.NewLogger(txId),
			})
			postProcessor.Submit(result)
			committer.Submit(result.StateChange)
			concurrent.TrySend(parallelStateChanges, result.StateChange)
		}
	})
	sequentialStateDB := &OperationLoggingStateDB{
		this.newStateDBForReading(),
		conflictDetector.NewLogger("sequential_set"),
	}
	this.applyBlockRewards(sequentialStateDB)
	committer.Submit(this.commitAsObject(sequentialStateDB))
	sequentialStateDB.CheckPoint(true)
	for i := 0; i < cap(parallelStateChanges); i++ {
		stateChange := <-parallelStateChanges
		this.err.CheckIn()
		sequentialStateDB.Merge(stateChange)
		sequentialStateDB.CheckPoint(true)
	}
	parallelTxSyncMeter()
	this.SequentialTransactions.Each(func(_ int, value interface{}) {
		defer this.metrics.SequentialTransactions.Recorder()()
		result := this.executeTransaction(value.(TxId), sequentialStateDB)
		postProcessor.Submit(result)
		committer.Submit(result.StateChange)
	})
	conflictDetector.Halt()
	this.metrics.TrieCommitSync.RecordElapsedTime(func() {
		this.result.StateRoot, _ = committer.AwaitResult()
	})
	this.metrics.ConflictDetectionSync.RecordElapsedTime(func() {
		conflictDetector.AwaitResult()
	})
	this.metrics.PostProcessingSync.RecordElapsedTime(func() {
		this.result.StateTransitionReceipt, _ = postProcessor.AwaitResult()
	})
	this.err.CheckIn()
	this.persistentCommit(this.result.StateRoot)
	return
}

func (this *stateTransition) RunLikeEthereum() (ret *StateTransitionResult, totalTime *metric_utils.AtomicCounter, err error) {
	util.Recover(this.err.Catch(util.SetTo(&err)))
	totalTime = new(metric_utils.AtomicCounter)
	ret = new(StateTransitionResult)
	defer totalTime.Recorder()()
	if this.Block.Number.Sign() == 0 {
		ret.StateRoot = this.applyGenesisBlock()
		return
	}
	stateDB := this.newStateDBForReading()
	for txId, tx := range this.Block.Transactions {
		txResult := this.VM.executeTransaction(&transactionRequest{
			txId:             txId,
			txData:           tx,
			blockHeader:      &this.Block.BlockHeader,
			stateDB:          stateDB,
			onEvmInstruction: vm.NoopExecutionController,
			gasPool:          new(core.GasPool).AddGas(this.Block.GasLimit),
			checkNonce:       true,
			metrics:          new(TransactionMetrics),
			canTransfer:      core.CanTransfer,
		})
		this.err.CheckIn(txResult.ConsensusErr)
		ret.UsedGas += txResult.GasUsed
		ethReceipt := types.NewReceipt(this.commitToTrie(stateDB), txResult.ContractErr != nil, ret.UsedGas)
		txData := this.Block.Transactions[txId]
		if txData.To == nil {
			ethReceipt.ContractAddress = crypto.CreateAddress(txData.From, txData.Nonce)
		}
		ethReceipt.TxHash = txData.Hash;
		ethReceipt.GasUsed = txResult.GasUsed
		ethReceipt.Logs = txResult.Logs
		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
		ret.Receipts = append(ret.Receipts, &TaraxaReceipt{
			ReturnValue:     txResult.EVMReturnValue,
			ContractError:   txResult.ContractErr,
			EthereumReceipt: ethReceipt,
		})
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}
	this.applyBlockRewards(stateDB)
	ret.StateRoot = this.commitToTrieAndCommitTrie(stateDB)
	this.persistentCommit(ret.StateRoot)
	return
}

type TestModeParams struct {
	DoCommitsInSeparateDB bool     `json:"doCommitsInSeparateDB"`
	DoCommits             bool     `json:"doCommits"`
	CommitSync            bool     `json:"commitSync"`
	SequentialTx          *TxIdSet `json:"sequentialTx"`
}

type TestModeTxMetrics struct {
	TotalTime   metric_utils.AtomicCounter `json:"totalTime"`
	LocalCommit metric_utils.AtomicCounter `json:"localCommit"`
	CreateDB    metric_utils.AtomicCounter `json:"createDB"`
}

type CommitterMetrics struct {
	ActualCommits metric_utils.AtomicCounter `json:"actualCommits"`
	TotalLifespan metric_utils.AtomicCounter `json:"totalLifespan"`
	CreateDB      metric_utils.AtomicCounter `json:"createDB"`
}

type MainExecutionMetrics struct {
	TotalTime        metric_utils.AtomicCounter `json:"totalTime"`
	TransactionsSync metric_utils.AtomicCounter `json:"transactionsSync"`
	CommitsSync      metric_utils.AtomicCounter `json:"commitsSync"`
}

type TestModeMetrics struct {
	Main               MainExecutionMetrics `json:"main"`
	Committer          CommitterMetrics     `json:"committer"`
	TransactionMetrics []TestModeTxMetrics  `json:"transactions"`
}

func (this *stateTransition) TestMode(params *TestModeParams) (metrics *TestModeMetrics) {
	metrics = new(TestModeMetrics)
	defer metrics.Main.TotalTime.Recorder()()
	txCount := len(this.Block.Transactions)
	metrics.TransactionMetrics = make([]TestModeTxMetrics, txCount)
	var committer *StateDBCommitter
	if txCount > 0 && (params.DoCommits || params.DoCommitsInSeparateDB) {
		commitsLeft := txCount
		recLifeSpan := metrics.Committer.TotalLifespan.Recorder()
		committer = LaunchStateDBCommitter(txCount, func() StateDB {
			defer metrics.Committer.CreateDB.Recorder()()
			if !params.DoCommitsInSeparateDB {
				return this.newStateDBForReading()
			}
			stateDB, err := state.New(this.BaseStateRoot, this.writeDB)
			this.err.SetIfAbsent(err)
			return stateDB
		}, func(db StateDB) common.Hash {
			if commitsLeft -= 1; commitsLeft == 0 {
				defer recLifeSpan()
			}
			defer metrics.Committer.ActualCommits.Recorder()()
			return this.commitToTrieAndCommitTrie(db)
		})
	}
	var syncCommitLock sync.Mutex
	allDone := concurrent.NewRendezvous(txCount)
	runTx := func(txId TxId, db StateDB) {
		txMetrics := &metrics.TransactionMetrics[txId]
		defer txMetrics.TotalTime.Recorder()()
		defer allDone.CheckIn()
		r := this.VM.executeTransaction(&transactionRequest{
			txId:             txId,
			txData:           this.Block.Transactions[txId],
			blockHeader:      &this.Block.BlockHeader,
			stateDB:          db,
			onEvmInstruction: vm.NoopExecutionController,
			gasPool:          new(core.GasPool).AddGas(eth_math.MaxUint64),
			checkNonce:       false,
			metrics:          new(TransactionMetrics),
			canTransfer: func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
				return true
			},
		})
		this.err.CheckIn(r.ConsensusErr)
		if committer != nil {
			defer txMetrics.LocalCommit.Recorder()()
			committer.Submit(this.commitAsObject(db))
		} else if params.CommitSync {
			syncCommitLock.Lock()
			defer syncCommitLock.Unlock()
			defer metrics.Committer.ActualCommits.Recorder()()
			defer metrics.Main.CommitsSync.Recorder()()
			defer txMetrics.LocalCommit.Recorder()()
			this.commitToTrieAndCommitTrie(db)
		}
	}
	recordTransactionSyncTime := metrics.Main.TransactionsSync.Recorder()
	sequentialTx := params.SequentialTx
	if sequentialTx == nil {
		sequentialTx = NewTxIdSet(nil)
	}
	if txCount != sequentialTx.Size() {
		lastScheduledTxId := int32(-1)
		parallelismFactor := 1.3 // Good
		numCPU := runtime.NumCPU()
		threadCount := int(float64(numCPU) * parallelismFactor)
		for i := 0; i < threadCount; i++ {
			go func() {
				stateDB := this.newStateDBForReading()
				for {
					txId := TxId(atomic.AddInt32(&lastScheduledTxId, 1))
					if txId >= txCount {
						break
					}
					if sequentialTx.Contains(txId) {
						continue
					}
					this.applyHardForks(stateDB)
					runTx(txId, stateDB)
					stateDB.Reset()
				}
			}()
		}
	}
	var sequentialStateDB StateDB = nil
	sequentialTx.Each(func(_ int, i interface{}) {
		if sequentialStateDB == nil {
			sequentialStateDB = this.newStateDBForReading()
		}
		runTx(i.(TxId), sequentialStateDB)
	})
	allDone.Await()
	recordTransactionSyncTime()
	if committer != nil {
		defer metrics.Main.CommitsSync.Recorder()()
		committer.AwaitResult()
	}
	this.err.CheckIn()
	return
}

func (this *stateTransition) executeTransaction(txId TxId, db StateDB) *TransactionResultWithStateChange {
	block := this.Block
	result := this.VM.executeTransaction(&transactionRequest{
		txId:        txId,
		txData:      block.Transactions[txId],
		blockHeader: &block.BlockHeader,
		stateDB:     db,
		gasPool:     new(core.GasPool).AddGas(block.GasLimit),
		checkNonce:  false,
		metrics:     &this.metrics.TransactionMetrics[txId],
		canTransfer: core.CanTransfer,
		onEvmInstruction: func(pc uint64) (uint64, bool) {
			this.err.CheckIn()
			return pc, true
		},
	})
	return &TransactionResultWithStateChange{result, this.commitAsObject(db)}
}

func (this *stateTransition) applyGenesisBlock() common.Hash {
	_, _, err := this.ApplyGenesis()
	this.err.CheckIn(err)
	return this.GenesisRoot()
}

func (this *stateTransition) applyHardForks(stateDB StateDB) (stateChanged bool) {
	chainConfig := this.Genesis.Config
	DAOForkBlock := chainConfig.DAOForkBlock
	if chainConfig.DAOForkSupport && DAOForkBlock != nil && DAOForkBlock.Cmp(this.Block.Number) == 0 {
		misc.ApplyDAOHardFork(stateDB.(*state.StateDB))
		return true
	}
	return false
}

func (this *stateTransition) applyBlockRewards(stateDB StateDB) {
	if this.DisableEthereumBlockReward {
		return
	}
	AccumulateMiningRewards(this.Genesis.Config, stateDB.(*state.StateDB),
		&this.Block.BlockNumberAndCoinbase, this.Block.Uncles...)
}

func (this *stateTransition) newStateDBForReading() StateDB {
	ret, err := state.New(this.BaseStateRoot, this.ReadDB)
	this.err.CheckIn(err)
	return ret
}

func (this *stateTransition) commitToTrie(db StateDB) (root []byte) {
	blockNumber := this.Block.Number
	chainConfig := this.Genesis.Config
	if chainConfig.IsByzantium(blockNumber) {
		db.Finalise(true)
	} else {
		root = db.IntermediateRoot(chainConfig.IsEIP158(blockNumber)).Bytes()
	}
	return
}

func (this *stateTransition) commitToTrieAndCommitTrie(db StateDB) common.Hash {
	defer this.metrics.TrieCommitTotal.Recorder()()
	ret, err := db.Commit(this.Genesis.Config.IsEIP158(this.Block.Number))
	this.err.CheckIn(err)
	return ret
}

// TODO make public
func (this *stateTransition) persistentCommit(root common.Hash) {
	defer this.metrics.PersistentCommit.Recorder()()
	trieDB := this.writeDB.TrieDB()
	defer trieDB.SetDiskDB(trieDB.GetDiskDB())
	trieDB.SetDiskDB(this.WriteDiskDB)
	this.err.CheckIn(trieDB.Commit(root, false))
}

func (this *stateTransition) commitAsObject(db StateDB) state.StateChange {
	return db.CommitStateChange(this.Genesis.Config.IsEIP158(this.Block.Number))
}
