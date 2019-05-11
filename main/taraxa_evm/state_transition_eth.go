package taraxa_evm

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/state_db_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type TransactionMetrics struct {
	TotalExecutionTime metric_utils.AtomicCounter `json:"totalExecutionTime"`
	TrieReads          metric_utils.AtomicCounter `json:"trieReads"`
	PersistentReads    metric_utils.AtomicCounter `json:"persistentReads"`
}

type Metrics struct {
	TransactionMetrics []*TransactionMetrics      `json:"transactionMetrics"`
	TrieCommitSync     metric_utils.AtomicCounter `json:"trieCommitSync"`
	TrieCommitTotal    metric_utils.AtomicCounter `json:"trieCommitTotal"`
	PersistentCommit   metric_utils.AtomicCounter `json:"persistentCommit"`
	TotalTime          metric_utils.AtomicCounter `json:"totalTime"`
}

func (this stateTransition) RunLikeEthereum() (ret *api.StateTransitionResult, metrics Metrics, err error) {
	metrics.TransactionMetrics = make([]*TransactionMetrics, len(this.Block.Transactions))
	readTrieDB := this.StateDB.TrieDB()
	readDiskDB := readTrieDB.GetDiskDB()
	readDiskDBPRoxy := &ethdb_proxy.DatabaseProxy{readDiskDB, new(proxy.Decorators)}
	defer readTrieDB.SetDiskDB(readDiskDB)
	readTrieDB.SetDiskDB(readDiskDBPRoxy)
	dbProxy := &state_db_proxy.DatabaseProxy{this.StateDB, new(proxy.Decorators), new(proxy.Decorators)}

	recordTotalTime := metrics.TotalTime.NewTimeRecorder()
	///
	//TODO remove
	defer util.Recover(this.err.Catch(util.SetTo(&err)))
	ret = new(api.StateTransitionResult)
	block := this.Block
	blockNumber := block.Number
	if blockNumber.Sign() == 0 {
		_, _, genesisSetupErr := core.SetupGenesisBlock(this.WriteDB, this.Genesis)
		this.err.CheckIn(genesisSetupErr)
		ret.StateRoot = this.Genesis.ToBlock(nil).Root()
		return
	}
	chainConfig := this.Genesis.Config

	comitter := LaunchStateDBCommitter(len(block.Transactions)+1, this.BaseStateRoot, dbProxy, &this.err,
		func(db StateDB) (common.Hash, error) {
			rec := metrics.TrieCommitTotal.NewTimeRecorder()
			defer rec()
			return db.Commit(chainConfig.IsEIP158(blockNumber))
		})

	stateDB, stateDbCreateErr := state.New(this.BaseStateRoot, dbProxy)
	this.err.CheckIn(stateDbCreateErr)
	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(blockNumber) == 0 {
		misc.ApplyDAOHardFork(stateDB)
	}
	ethash.AccumulateRewards(chainConfig, stateDB, &block.HeaderNumerAndCoinbase, block.Uncles...)

	comitter.RequestCommit(stateDB.CommitLocally())

	gasPool := new(core.GasPool).AddGas(block.GasLimit)
	for txId := range block.Transactions {
		var taraxaReceipt *api.TaraxaReceipt

		txMetrics := new(TransactionMetrics)
		readDiskDBPRoxy.Decorators.Register("Get", metric_utils.MeasureElapsedTime(&txMetrics.PersistentReads))
		readDiskDBPRoxy.Decorators.Register("Has", metric_utils.MeasureElapsedTime(&txMetrics.PersistentReads))
		dbProxy.Decorators.Register("OpenTrie", metric_utils.MeasureElapsedTime(&txMetrics.TrieReads))
		dbProxy.Decorators.Register("OpenStorageTrie", metric_utils.MeasureElapsedTime(&txMetrics.TrieReads))
		dbProxy.Decorators.Register("ContractCode", metric_utils.MeasureElapsedTime(&txMetrics.TrieReads))
		dbProxy.TrieDecorators.Register("TryGet", metric_utils.MeasureElapsedTime(&txMetrics.TrieReads))
		txMetrics.TotalExecutionTime.MeasureElapsedTime(func() {
			taraxaReceipt = this.executeTransaction(txId, stateDB, gasPool, true)
		})
		readDiskDBPRoxy.Decorators.Delete("Get", "Has")
		dbProxy.Decorators.Delete("OpenTrie", "OpenStorageTrie", "ContractCode")
		dbProxy.TrieDecorators.Delete("TryGet")
		metrics.TransactionMetrics[txId] = txMetrics

		comitter.RequestCommit(stateDB.CommitLocally())

		ethReceipt := taraxaReceipt.EthereumReceipt
		//metrics.TrieCommit.MeasureElapsedTime(func() {
		//	ethReceipt.PostState = this.commitTransaction(stateDB)
		//})
		ret.UsedGas += ethReceipt.GasUsed
		ethReceipt.CumulativeGasUsed = ret.UsedGas
		ret.Receipts = append(ret.Receipts, taraxaReceipt)
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}

	metrics.TrieCommitSync.MeasureElapsedTime(func() {
		ret.StateRoot = comitter.AwaitFinalRoot()
	})

	util.Assert(ret.StateRoot == this.ExpectedRoot, ret.StateRoot.Hex(), " != ", this.ExpectedRoot.Hex())
	metrics.PersistentCommit.MeasureElapsedTime(func() {
		finalCommitErr := this.PersistentCommit(ret.StateRoot)
		this.err.CheckIn(finalCommitErr)
	})

	recordTotalTime()

	trieCommitSpeedup := float64(metrics.TrieCommitTotal) / float64(metrics.TrieCommitSync)
	fmt.Println("trie commit speedup: ", trieCommitSpeedup)

	totalSpeedup := float64(metrics.TotalTime-metrics.TrieCommitSync+metrics.TrieCommitTotal) / float64(metrics.TotalTime)
	fmt.Println("total speedup: ", totalSpeedup)

	return
}
