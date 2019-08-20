package taraxa_vm

import (
	"errors"
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
	"github.com/Taraxa-project/taraxa-evm/taraxa/taraxa_types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/rendezvous"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
)

type TransactionMetrics struct {
	TotalTime metric_utils.AtomicCounter `json:"totalTime"`
	//TrieReads          metric_utils.AtomicCounter `json:"trieReads"`
	//PersistentReads    metric_utils.AtomicCounter `json:"persistentReads"`
}

type ScheduleGenerationMetrics struct {
	TransactionMetrics     []TransactionMetrics       `json:"transactionMetrics"`
	TotalTime              metric_utils.AtomicCounter `json:"totalTime"`
	ConflictPostProcessing metric_utils.AtomicCounter `json:"conflictPostProcessing"`
}

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
	*TaraxaVM
	*taraxa_types.StateTransitionRequest
	*taraxa_types.ConcurrentSchedule
	metrics StateTransitionMetrics
	err     util.ErrorBarrier
}

func (this *stateTransition) run() (ret *taraxa_types.StateTransitionResult, metrics *StateTransitionMetrics, err error) {
	//defer util.Recover(this.err.Catch(util.SetTo(&err)))
	this.metrics.TransactionMetrics = make([]TransactionMetrics, len(this.Block.Transactions))
	metrics = &this.metrics
	ret = new(taraxa_types.StateTransitionResult)
	block := this.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		ret.StateRoot = this.applyGenesisBlock()
		return
	}
	txCount := len(block.Transactions)
	sequentialTransactions := util.NewLinkedHashSet(this.SequentialTransactions)
	parallelTxCount := txCount - sequentialTransactions.Size()
	conflictDetectorInboxCapacity := (parallelTxCount + 1) * this.ConflictDetectorInboxPerTransaction
	conflictDetector := conflict_detector.New(conflictDetectorInboxCapacity, this.onConflict)
	go conflictDetector.Run()
	defer conflictDetector.Halt()
	committer := LaunchStateDBCommitter(txCount+1, this.newStateDBForReading, this.commitToTrie)
	defer committer.SignalShutdown()
	postProcessor := LaunchBlockPostProcessor(block, this.newStateDBForReading, func(err error) {
		this.err.SetIfAbsent(err)
	})
	defer postProcessor.SignalShutdown()

	defer metrics.TotalTime.NewTimeRecorder()()

	parallelTxSyncMeter := this.metrics.ParallelTransactionsSync.NewTimeRecorder()

	parallelStateChanges := make(chan state.StateChange, parallelTxCount)
	for txId := range this.Block.Transactions {
		if sequentialTransactions.Contains(txId) {
			continue
		}
		txId := txId
		go func() {
			defer util.Recover(this.err.Catch(func(error) {
				util.TryClose(parallelStateChanges)
			}))
			db := &OperationLoggingStateDB{this.newStateDBForReading(), conflictDetector.NewLogger(txId)}
			result := this.executeTransaction(txId, db)
			postProcessor.Submit(result)
			committer.Submit(result.StateChange)
			util.TrySend(parallelStateChanges, result.StateChange)
		}()
	}
	sequentialStateDB := this.newStateDBForReading()
	// TODO move somewhere else
	this.applyBlockRewards(sequentialStateDB)
	committer.Submit(this.commitAsObject(sequentialStateDB))
	for i := 0; i < cap(parallelStateChanges); i++ {
		stateChange := <-parallelStateChanges
		this.err.CheckIn()
		sequentialStateDB.Merge(stateChange)
		sequentialStateDB.CheckPoint(true)
	}

	parallelTxSyncMeter()

	conflictDetector.Halt()
	sequentialTransactions.Each(func(_ int, value interface{}) {
		defer this.metrics.SequentialTransactions.NewTimeRecorder()()
		result := this.executeTransaction(value.(taraxa_types.TxId), sequentialStateDB)
		postProcessor.Submit(result)
		committer.Submit(result.StateChange)
	})
	metrics.TrieCommitSync.MeasureElapsedTime(func() {
		ret.StateRoot, _ = committer.AwaitResult()
		this.err.CheckIn()
	})
	this.metrics.ConflictDetectionSync.MeasureElapsedTime(func() {
		conflictDetector.AwaitResult()
		this.err.CheckIn()
	})
	this.metrics.PostProcessingSync.MeasureElapsedTime(func() {
		//ret.StateTransitionReceipt, _ = postProcessor.AwaitResult()
		this.err.CheckIn()
	})
	//util.Assert(ret.StateRoot == this.ExpectedRoot, ret.StateRoot.Hex(), " != ", this.ExpectedRoot.Hex())
	this.persistentCommit(ret.StateRoot)
	return
}

func (this *stateTransition) RunLikeEthereum() (ret *taraxa_types.StateTransitionResult, totalTime *metric_utils.AtomicCounter, err error) {
	util.Recover(this.err.Catch(util.SetTo(&err)))
	totalTime = new(metric_utils.AtomicCounter)
	ret = new(taraxa_types.StateTransitionResult)
	defer totalTime.NewTimeRecorder()()
	if this.Block.Number.Sign() == 0 {
		ret.StateRoot = this.applyGenesisBlock()
		return
	}
	stateDB := this.newStateDBForReading()
	this.applyHardForks(stateDB)
	gp := new(core.GasPool).AddGas(this.Block.GasLimit)
	for txId, tx := range this.Block.Transactions {
		txResult := this.TaraxaVM.executeTransaction(&transactionRequest{
			txId,
			tx,
			&this.Block.BlockHeader,
			stateDB,
			vm.NoopExecutionController,
			gp,
			true,
			new(TransactionMetrics),
			core.CanTransfer,
		})
		this.err.CheckIn(txResult.ConsensusErr)
		intermediateRoot := this.commitToTrie(stateDB)
		var intermediateRootBytes []byte
		if !this.Genesis.Config.IsByzantium(this.Block.Number) {
			intermediateRootBytes = intermediateRoot.Bytes()
		}
		ret.UsedGas += txResult.GasUsed
		ethReceipt := types.NewReceipt(intermediateRootBytes, txResult.ContractErr != nil, ret.UsedGas)
		txData := this.Block.Transactions[txId]
		if txData.To == nil {
			ethReceipt.ContractAddress = crypto.CreateAddress(txData.From, txData.Nonce)
		}
		ethReceipt.TxHash = txData.Hash;
		ethReceipt.GasUsed = txResult.GasUsed
		ethReceipt.Logs = txResult.Logs
		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
		ret.Receipts = append(ret.Receipts, &taraxa_types.TaraxaReceipt{
			ReturnValue:     txResult.EVMReturnValue,
			ContractError:   txResult.ContractErr,
			EthereumReceipt: ethReceipt,
		})
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}
	if !this.DisableEthereumBlockReward {
		this.applyBlockRewards(stateDB)
	}
	ret.StateRoot = this.commitToTrie(stateDB)
	this.persistentCommit(ret.StateRoot)
	return
}

type TestModeParams struct {
	DoCommitsInSeparateDB bool                  `json:"doCommitsInSeparateDB"`
	DoCommits             bool                  `json:"doCommits"`
	CommitSync            bool                  `json:"commitSync"`
	SequentialTx          *taraxa_types.TxIdSet `json:"sequentialTx"`
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
	defer metrics.Main.TotalTime.NewTimeRecorder()()
	txCount := len(this.Block.Transactions)
	metrics.TransactionMetrics = make([]TestModeTxMetrics, txCount)
	var committer *StateDBCommitter
	if txCount > 0 && (params.DoCommits || params.DoCommitsInSeparateDB) {
		commitsLeft := txCount
		recLifeSpan := metrics.Committer.TotalLifespan.NewTimeRecorder()
		committer = LaunchStateDBCommitter(txCount, func() StateDB {
			defer metrics.Committer.CreateDB.NewTimeRecorder()()
			if !params.DoCommitsInSeparateDB {
				return this.newStateDBForReading()
			}
			stateDB, err := state.New(this.BaseStateRoot, this.WriteDB)
			this.err.SetIfAbsent(err)
			return stateDB
		}, func(db StateDB) common.Hash {
			if commitsLeft -= 1; commitsLeft == 0 {
				defer recLifeSpan()
			}
			defer metrics.Committer.ActualCommits.NewTimeRecorder()()
			return this.commitToTrie(db)
		})
	}
	var syncCommitLock sync.Mutex
	allDone := rendezvous.New(txCount)
	runTx := func(txId taraxa_types.TxId, db StateDB) {
		txMetrics := &metrics.TransactionMetrics[txId]
		defer txMetrics.TotalTime.NewTimeRecorder()()
		defer allDone.CheckIn()
		r := this.TaraxaVM.executeTransaction(&transactionRequest{
			txId,
			this.Block.Transactions[txId],
			&this.Block.BlockHeader,
			db,
			vm.NoopExecutionController,
			new(core.GasPool).AddGas(eth_math.MaxUint64),
			false,
			new(TransactionMetrics),
			func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
				return true
			},
		})
		this.err.CheckIn(r.ConsensusErr)
		if committer != nil {
			defer txMetrics.LocalCommit.NewTimeRecorder()()
			committer.Submit(this.commitAsObject(db))
		} else if params.CommitSync {
			syncCommitLock.Lock()
			defer syncCommitLock.Unlock()
			defer metrics.Committer.ActualCommits.NewTimeRecorder()()
			defer metrics.Main.CommitsSync.NewTimeRecorder()()
			defer txMetrics.LocalCommit.NewTimeRecorder()()
			this.commitToTrie(db)
		}
	}
	recordTransactionSyncTime := metrics.Main.TransactionsSync.NewTimeRecorder()
	sequentialTx := params.SequentialTx
	if sequentialTx == nil {
		sequentialTx = taraxa_types.NewTxIdSet(nil)
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
					txId := taraxa_types.TxId(atomic.AddInt32(&lastScheduledTxId, 1))
					if txId >= txCount {
						break
					}
					if sequentialTx.Contains(txId) {
						continue
					}
					runTx(txId, stateDB)
					stateDB.(*state.StateDB).Reset()
					//runtime.Gosched()
				}
			}()
		}
	}
	var sequentialStateDB StateDB = nil
	sequentialTx.Each(func(_ int, i interface{}) {
		if sequentialStateDB == nil {
			sequentialStateDB = this.newStateDBForReading()
		}
		runTx(i.(taraxa_types.TxId), sequentialStateDB)
	})
	allDone.Await()
	recordTransactionSyncTime()
	if committer != nil {
		defer metrics.Main.CommitsSync.NewTimeRecorder()()
		committer.AwaitResult()
	}
	this.err.CheckIn()
	return
}

func (this *stateTransition) executeTransaction(txId taraxa_types.TxId, db StateDB) *TransactionResultWithStateChange {
	block := this.Block
	result := this.TaraxaVM.executeTransaction(&transactionRequest{
		txId,
		block.Transactions[txId],
		&block.BlockHeader,
		db,
		this.onEvmInstruction,
		new(core.GasPool).AddGas(block.GasLimit),
		false,
		&this.metrics.TransactionMetrics[txId],
		core.CanTransfer,
		//func(vm.StateDB, common.Address, *big.Int) bool {
		//	return true
		//},
	})
	stateChange := this.commitAsObject(db)
	return &TransactionResultWithStateChange{result, stateChange}
}

func (this *stateTransition) onEvmInstruction(programCounter uint64) (programCounterChanged uint64, canProceed bool) {
	this.err.CheckIn()
	return programCounter, true
}

func (this *stateTransition) onConflict(_ *conflict_detector.ConflictDetector, op *conflict_detector.Operation, authors conflict_detector.Authors) {
	this.err.SetIfAbsent(errors.New(
		//fmt.Sprintf("Conflict detected. Operation: {%s, %s, %s}; Authors: %s", op.Author, op.Type, op.Key, util.Join(", ", authors.Values())),
		"Conflict detected: " + util.Join(", ", authors.Values()),
	))
}

func (this *stateTransition) applyGenesisBlock() common.Hash {
	_, _, genesisSetupErr := core.SetupGenesisBlock(this.WriteDiskDB, this.Genesis)
	this.err.CheckIn(genesisSetupErr)
	return this.Genesis.ToBlock(nil).Root()
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
	AccumulateRewards(this.Genesis.Config, stateDB.(*state.StateDB),
		&this.Block.BlockNumberAndCoinbase, this.Block.Uncles...)
}

func (this *stateTransition) newStateDBForReading() StateDB {
	stateDB, err := state.New(this.BaseStateRoot, this.ReadDB)
	this.err.CheckIn(err)
	util.Assert(!this.applyHardForks(stateDB))
	return stateDB
}

func (this *stateTransition) commitAsObject(db StateDB) state.StateChange {
	return db.CommitStateChange(this.Genesis.Config.IsEIP158(this.Block.Number))
}

func (this *stateTransition) commitToTrie(db StateDB) common.Hash {
	defer this.metrics.TrieCommitTotal.NewTimeRecorder()()
	root, err := db.Commit(this.Genesis.Config.IsEIP158(this.Block.Number))
	this.err.SetIfAbsent(err)
	return root
}

// TODO make public
func (this *stateTransition) persistentCommit(root common.Hash) {
	defer this.metrics.PersistentCommit.NewTimeRecorder()()
	trieDB := this.WriteDB.TrieDB()
	defer trieDB.SetDiskDB(trieDB.GetDiskDB())
	trieDB.SetDiskDB(this.WriteDiskDB)
	this.err.CheckIn(trieDB.Commit(root, false))
}

//func (this *stateTransition) commitToTrie(db *state.StateDB) (root common.Hash) {
//	blockNumber := this.Block.Number
//	chainConfig := this.Genesis.Config
//	if chainConfig.IsByzantium(blockNumber) {
//		db.Finalise(true)
//	} else {
//		root = db.IntermediateRoot(chainConfig.IsEIP158(blockNumber))
//	}
//	return
//}
