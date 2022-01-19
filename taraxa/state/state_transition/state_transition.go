package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/hardfork"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type StateTransition struct {
	chain_config       *chain_config.ChainConfig
	state              state_db.LatestState
	pending_blk_state  state_db.PendingBlockState
	evm_state          state_evm.EVMState
	evm                vm.EVM
	trie_sink          TrieSink
	pending_state_root common.Hash
	dpos_contract      *dpos.Contract
	get_reader         func(types.BlockNum) dpos.Reader
	new_chain_config   *chain_config.ChainConfig
}
type StateTransitionConfig struct {
	ETHChainConfig      params.ChainConfig
	DisableBlockRewards bool
	ExecutionOptions    vm.ExecutionOpts
	GenesisBalances     core.BalanceMap
}
type Opts struct {
	EVMState state_evm.Opts
	Trie     TrieSinkOpts
}

func (self *StateTransition) Init(
	state state_db.LatestState,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	get_reader func(types.BlockNum) dpos.Reader,
	chain_config *chain_config.ChainConfig,
	opts Opts,
) *StateTransition {
	self.chain_config = chain_config
	self.state = state
	self.evm_state.Init(opts.EVMState)
	self.get_reader = get_reader
	self.evm.Init(get_block_hash, &self.evm_state, vm.Opts{
		// 24MB total
		U256PoolSize:           32 * vm.StackLimit,
		NumStacksToPreallocate: vm.StackLimit,
		PreallocatedStackSize:  vm.StackLimit,
		PreallocatedMem:        8 * 1024 * 1024,
	})
	state_desc := state.GetCommittedDescriptor()
	self.trie_sink.Init(&state_desc.StateRoot, opts.Trie)
	if dpos_api != nil {
		self.dpos_contract = dpos_api.NewContract(dpos.EVMStateStorage{&self.evm_state})
	}
	if state_common.IsEmptyStateRoot(&state_desc.StateRoot) {
		self.begin_block()
		asserts.Holds(self.pending_blk_state.GetNumber() == 0)
		for addr, balance := range self.chain_config.GenesisBalances {
			self.evm_state.GetAccount(&addr).AddBalance(balance)
		}
		if self.dpos_contract != nil {
			util.PanicIfNotNil(self.dpos_contract.ApplyGenesis())
		}
		self.evm_state_checkpoint()
		self.Commit()
	}
	// we need genesis balances later, so it is commented
	// self.chain_config.GenesisBalances = nil
	return self
}

func (self *StateTransition) UpdateConfig(cfg *chain_config.ChainConfig) {
	self.new_chain_config = cfg
}

func (self *StateTransition) Close() {
	self.trie_sink.Close()
}

func (self *StateTransition) begin_block() {
	self.pending_blk_state = self.state.BeginPendingBlock()
	self.evm_state.SetInput(state_db.ExtendedReader{self.pending_blk_state})
	self.trie_sink.SetIO(self.pending_blk_state)
}

func (self *StateTransition) evm_state_checkpoint() {
	self.evm_state.CommitTransaction(&self.trie_sink, self.evm.GetRules().IsEIP158)
}

func (self *StateTransition) BeginBlock(blk_info *vm.BlockInfo) {
	self.begin_block()
	blk_n := self.pending_blk_state.GetNumber()
	rules_changed := self.evm.SetBlock(&vm.Block{blk_n, *blk_info}, self.chain_config.ETHChainConfig.Rules(blk_n))
	if self.dpos_contract != nil && rules_changed {
		self.dpos_contract.Register(self.evm.RegisterPrecompiledContract)
	}
	if self.chain_config.ETHChainConfig.IsDAOFork(blk_n) {
		misc.ApplyDAOHardFork(&self.evm_state)
		self.evm_state_checkpoint()
	}
	if self.chain_config.Hardforks.IsFixGenesisFork(blk_n) {
		if self.new_chain_config == nil {
			panic("we should have new_chain_config for hardfork")
		}
		self.chain_config = self.new_chain_config
		self.new_chain_config = nil

		self.dpos_contract.ResetGenesisAddresses(self.chain_config.DPOS.GenesisState)
		self.dpos_contract.UpdateConfig(*self.chain_config.DPOS)
		hardfork.ApplyFixGenesisFork(self.chain_config.GenesisBalances, self.chain_config.DPOS, &self.evm_state, self.dpos_contract)
	}
}

func (self *StateTransition) ExecuteTransaction(trx *vm.Transaction) (ret vm.ExecutionResult) {
	ret = self.evm.Main(trx, self.chain_config.ExecutionOptions)
	self.evm_state_checkpoint()
	return
}

func (self *StateTransition) EndBlock(uncles []state_common.UncleBlock) {
	if self.dpos_contract != nil {
		self.dpos_contract.Commit(self.pending_blk_state.GetNumber())
		self.evm_state_checkpoint()
	}
	if !self.chain_config.DisableBlockRewards {
		evm_block := self.evm.GetBlock()
		ethash.AccumulateRewards(
			self.evm.GetRules(),
			ethash.BlockNumAndCoinbase{evm_block.Number, evm_block.Author},
			uncles,
			&self.evm_state)
		self.evm_state_checkpoint()
	}
	self.pending_blk_state = nil
}

func (self *StateTransition) PrepareCommit() common.Hash {
	self.evm_state.Commit()
	self.evm_state.SetInput(nil)
	self.pending_state_root = self.trie_sink.Commit()
	self.trie_sink.SetIO(nil)
	return self.pending_state_root
}

func (self *StateTransition) Commit() (state_root common.Hash) {
	if self.pending_state_root == common.ZeroHash {
		self.PrepareCommit()
	}
	state_root, self.pending_state_root = self.pending_state_root, common.ZeroHash
	util.PanicIfNotNil(self.state.Commit(state_root)) // TODO move out of here, this should be async
	return
}
