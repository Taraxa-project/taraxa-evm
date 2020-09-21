package state_transition

import (
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_historical"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_config"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"

	"github.com/Taraxa-project/taraxa-evm/core"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

// TODO memory leaks??
type StateTransition struct {
	exec_cfg          state_config.ExecutionConfig
	db                state_common.DB
	dpos_contract     *dpos.Contract
	curr_blk_num      types.BlockNum
	exec_results      []vm.ExecutionResult
	evm               vm.EVM
	last_block_reader state_historical.BlockReader
	evm_state         state_evm.EVMState
	trie_sink         TrieSink
}

type StateTransitionOpts struct {
	TrieWriters               TrieWriterOpts
	ExpectedMaxNumTrxPerBlock uint32
}

func (self *StateTransition) Init(
	db state_common.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_config.ChainConfig,
	curr_blk_num types.BlockNum,
	curr_state_root *common.Hash,
	opts StateTransitionOpts,
) *StateTransition {
	self.db = db
	self.exec_cfg = chain_cfg.Execution
	self.curr_blk_num = curr_blk_num
	dirty_accs_per_block := uint32(util.CeilPow2(int(opts.ExpectedMaxNumTrxPerBlock * 2)))
	accs_per_block := dirty_accs_per_block * 2
	self.evm_state.Init(&self.last_block_reader, state_evm.CacheOpts{
		AccountBufferSize: accs_per_block * 2,
		RevertLogSize:     4 * 64,
	})
	self.evm.Init(vm.Config{
		U256PoolSize:        vm.StackLimit,
		NumStacksToPrealloc: vm.StackLimit,
		StackPrealloc:       vm.StackLimit,
		MemPoolSize:         32 * 1024 * 1024,
	})
	self.evm.SetGetHash(get_block_hash)
	self.evm.SetState(&self.evm_state)
	self.trie_sink.Init(curr_state_root, TrieSinkOpts{
		TrieWriters:              opts.TrieWriters,
		NumDirtyAccountsToBuffer: dirty_accs_per_block,
	})
	self.exec_results = make([]vm.ExecutionResult, opts.ExpectedMaxNumTrxPerBlock)
	if chain_cfg.DPOS != nil {
		self.dpos_contract = new(dpos.Contract).Init(*chain_cfg.DPOS, dpos_storage_adapter{self})
	}
	return self
}

type GenesisConfig struct {
	Balances core.BalanceMap
	DPOS     *dpos.GenesisConfig
}

func (self *StateTransition) GenesisInit(cfg GenesisConfig) *common.Hash {
	self.begin_block()
	for addr, balance := range cfg.Balances {
		self.evm_state.GetAccount(&addr).AddBalance(balance)
	}
	if self.dpos_contract != nil {
		util.PanicIfNotNil(self.dpos_contract.GenesisInit(*cfg.DPOS))
	}
	self.evm_state_checkpoint()
	batch := self.evm_state.Commit()
	defer self.evm_state.Cleanup(batch)
	return self.trie_sink.CommitSync(batch)
}

func (self *StateTransition) begin_block() {
	db_tx := self.db.NewBlockCreationTransaction(self.curr_blk_num)
	self.last_block_reader.SetTransaction(db_tx)
	self.trie_sink.BeginBatch(db_tx)
}

func (self *StateTransition) evm_state_checkpoint() {
	self.evm_state.Checkpoint(&self.trie_sink, self.evm.Rules.IsEIP158)
}

func (self *StateTransition) BeginBlock(blk *vm.BlockWithoutNumber) {
	self.curr_blk_num++
	self.begin_block()
	self.evm.SetBlock(self.curr_blk_num, blk)
	precompiles_changed := self.evm.SetRules(self.exec_cfg.ETHForks.Rules(self.curr_blk_num))
	if self.dpos_contract != nil && precompiles_changed {
		self.dpos_contract.Register(self.evm.RegisterPrecompiledContract)
	}
	if self.exec_cfg.ETHForks.IsDAOFork(self.curr_blk_num) {
		misc.ApplyDAOHardFork(&self.evm_state)
		self.evm_state_checkpoint()
	}
}

func (self *StateTransition) SubmitTransaction(trx *vm.Transaction) {
	self.exec_results = append(self.exec_results, self.evm.Main(trx, self.exec_cfg.Options))
	self.evm_state_checkpoint()
}

func (self *StateTransition) EndBlock(uncles []state_common.UncleBlock) {
	if self.dpos_contract != nil {
		self.dpos_contract.Commit()
		self.evm_state_checkpoint()
	}
	if !self.exec_cfg.DisableBlockRewards {
		ethash.AccumulateRewards(
			self.evm.Rules,
			ethash.BlockNumAndCoinbase{self.curr_blk_num, self.evm.Block.Author},
			uncles,
			&self.evm_state)
		self.evm_state_checkpoint()
	}
}

type StateTransitionResult struct {
	StateRoot        *common.Hash
	ExecutionResults []vm.ExecutionResult
}

func (self *StateTransition) CommitSync() (ret StateTransitionResult) {
	ret.ExecutionResults = self.exec_results
	bin.ZFill_3(unsafe.Pointer(&self.exec_results), unsafe.Sizeof(self.exec_results[0]))
	self.exec_results = self.exec_results[:0]
	batch := self.evm_state.Commit()
	defer self.evm_state.Cleanup(batch)
	ret.StateRoot = self.trie_sink.CommitSync(batch)
	return
}
