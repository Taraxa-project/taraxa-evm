package state_transition

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
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
	LastBlockNum       uint64
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
		self.dpos_contract = dpos_api.NewContract(dpos.EVMStateStorage{&self.evm_state}, get_reader(state_desc.BlockNum))
	}
	if state_common.IsEmptyStateRoot(&state_desc.StateRoot) {
		self.begin_block()
		asserts.Holds(self.pending_blk_state.GetNumber() == 0)
		for addr, balance := range self.chain_config.GenesisBalances {
			self.evm_state.GetAccount(&addr).AddBalance(balance)
		}
		if self.dpos_contract != nil {
			util.PanicIfNotNil(self.dpos_contract.ApplyGenesis(self.evm_state.GetAccount))
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
}

func (self *StateTransition) ExecuteTransaction(tx *vm.Transaction) (ret vm.ExecutionResult) {
	ret = self.evm.Main(tx, self.chain_config.ExecutionOptions)
	self.evm_state_checkpoint()
	return
}

func (self *StateTransition) GetChainConfig() (ret *chain_config.ChainConfig) {
	ret = self.chain_config
	return
}

func (self *StateTransition) EndBlock(uncles []state_common.UncleBlock, rewardsStats *rewards_stats.RewardsStats, feesRewards *dpos.FeesRewards) {
	if !self.chain_config.BlockRewardsOptions.DisableBlockRewards {
		evm_block := self.evm.GetBlock()
		if self.chain_config.BlockRewardsOptions.DisableContractDistribution {
			ethash.AccumulateRewards(
				self.evm.GetRules(),
				ethash.BlockNumAndCoinbase{evm_block.Number, evm_block.Author},
				uncles,
				&self.evm_state)
		} else if rewardsStats != nil && feesRewards != nil {
			if self.dpos_contract == nil {
				panic("Stats rewards enabled but no dpos contract registered")
			}
			self.dpos_contract.DistributeRewards(rewardsStats, feesRewards)
		}
		self.evm_state_checkpoint()
	}
	self.LastBlockNum = self.evm.GetBlock().Number
	if self.dpos_contract != nil {
		self.dpos_contract.EndBlockCall()
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
	if self.dpos_contract != nil {
		self.dpos_contract.CommitCall(self.get_reader(self.evm.GetBlock().Number))
	}
	return
}

func (self *StateTransition) AddTxFeeToBalance(account *common.Address, tx_fee *big.Int) {
	self.evm_state.GetAccount(account).AddBalance(tx_fee)
}
