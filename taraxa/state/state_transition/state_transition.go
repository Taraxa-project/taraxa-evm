package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

// TODO memory leaks??
type StateTransition struct {
	exec_cfg          state_common.ExecutionConfig
	state             state_db.LatestState
	pending_blk_state state_db.PendingBlockState
	evm_state         state_evm.EVMState
	evm               vm.EVM
	trie_sink         TrieSink
	dpos_contract     *dpos.Contract
}
type Opts struct {
	TrieWriters               TrieWriterOpts
	ExpectedMaxNumTrxPerBlock uint32
}

func (self *StateTransition) Init(
	state state_db.LatestState,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	exec_cfg state_common.ExecutionConfig,
	genesis_balances core.BalanceMap,
	opts Opts,
) *StateTransition {
	self.exec_cfg = exec_cfg
	self.state = state
	dirty_accs_per_block := uint32(util.CeilPow2(int(opts.ExpectedMaxNumTrxPerBlock * 2)))
	accs_per_block := dirty_accs_per_block * 2
	self.evm_state.Init(state_evm.CacheOpts{
		AccountBufferSize: accs_per_block * 2,
		RevertLogSize:     4 * 64,
	})
	self.evm.Init(get_block_hash, &self.evm_state, vm.Opts{
		U256PoolSize:        vm.StackLimit,
		NumStacksToPrealloc: vm.StackLimit,
		StackPrealloc:       vm.StackLimit,
		MemPoolSize:         32 * 1024 * 1024,
	})
	state_desc := state.GetCommittedDescriptor()
	self.trie_sink.Init(&state_desc.StateRoot, TrieSinkOpts{
		TrieWriters:              opts.TrieWriters,
		NumDirtyAccountsToBuffer: dirty_accs_per_block,
	})
	if dpos_api != nil {
		self.dpos_contract = dpos_api.NewContract(dpos.EVMStateStorage{&self.evm_state})
	}
	if state_common.IsEmptyStateRoot(&state_desc.StateRoot) {
		self.begin_block()
		assert.Holds(self.pending_blk_state.GetNumber() == 0)
		for addr, balance := range genesis_balances {
			self.evm_state.GetAccount(&addr).AddBalance(balance)
		}
		if self.dpos_contract != nil {
			util.PanicIfNotNil(self.dpos_contract.ApplyGenesis())
		}
		self.evm_state_checkpoint()
		self.Commit()
	}
	return self
}

func (self *StateTransition) begin_block() {
	self.pending_blk_state = self.state.BeginPendingBlock()
	self.evm_state.SetInput(state_db.ExtendedReader{self.pending_blk_state})
	self.trie_sink.SetIO(self.pending_blk_state)
}

func (self *StateTransition) evm_state_checkpoint() {
	self.evm_state.Checkpoint(&self.trie_sink, self.evm.Rules.IsEIP158)
}

func (self *StateTransition) BeginBlock(blk *vm.BlockInfo) {
	self.begin_block()
	blk_n := self.pending_blk_state.GetNumber()
	rules_changed := self.evm.SetBlock(blk_n, blk, self.exec_cfg.ETHForks.Rules(blk_n))
	if self.dpos_contract != nil && rules_changed {
		self.dpos_contract.Register(self.evm.RegisterPrecompiledContract)
	}
	if self.exec_cfg.ETHForks.IsDAOFork(blk_n) {
		misc.ApplyDAOHardFork(&self.evm_state)
		self.evm_state_checkpoint()
	}
}

func (self *StateTransition) ExecuteTransaction(trx *vm.Transaction) (ret vm.ExecutionResult) {
	ret = self.evm.Main(trx, self.exec_cfg.Options)
	self.evm_state_checkpoint()
	return
}

func (self *StateTransition) EndBlock(uncles []state_common.UncleBlock) {
	blk_n := self.pending_blk_state.GetNumber()
	if self.dpos_contract != nil {
		self.dpos_contract.Commit(blk_n)
		self.evm_state_checkpoint()
	}
	if !self.exec_cfg.DisableBlockRewards {
		ethash.AccumulateRewards(
			self.evm.Rules,
			ethash.BlockNumAndCoinbase{blk_n, self.evm.Block.Author},
			uncles,
			&self.evm_state)
		self.evm_state_checkpoint()
	}
	self.pending_blk_state = nil
}

func (self *StateTransition) Commit() (state_root *common.Hash) {
	self.evm_state.Commit()
	self.evm_state.SetInput(nil)
	state_root = self.trie_sink.CommitSync()
	self.trie_sink.SetIO(nil)
	util.PanicIfNotNil(self.state.Commit(state_root)) // TODO move out of here, this should be async
	return
}
