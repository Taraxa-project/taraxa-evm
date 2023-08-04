package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/holiman/uint256"
)

type StateTransition struct {
	chain_config        *chain_config.ChainConfig
	state               state_db.LatestState
	pending_blk_state   state_db.PendingBlockState
	evm_state           state_evm.EVMState
	evm                 vm.EVM
	trie_sink           TrieSink
	pending_state_root  common.Hash
	dpos_contract       *dpos.Contract
	get_dpos_reader     func(types.BlockNum) dpos.Reader
	slashing_contract   *slashing.Contract
	get_slashing_reader func(types.BlockNum) slashing.Reader
	new_chain_config    *chain_config.ChainConfig
	LastBlockNum        uint64
}

type Opts struct {
	EVMState state_evm.Opts
	Trie     TrieSinkOpts
}

func (self *StateTransition) Init(
	state state_db.LatestState,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	get_dpos_reader func(types.BlockNum) dpos.Reader,
	slashing_api *slashing.API,
	get_slashing_reader func(types.BlockNum) slashing.Reader,
	chain_config *chain_config.ChainConfig,
	opts Opts,
) *StateTransition {
	self.chain_config = chain_config
	self.state = state
	self.evm_state.Init(opts.EVMState)
	self.get_dpos_reader = get_dpos_reader
	self.get_slashing_reader = get_slashing_reader
	self.evm.Init(get_block_hash, &self.evm_state, vm.Opts{
		// 24MB total
		PreallocatedMem: 8 * 1024 * 1024,
	}, self.chain_config.EVMChainConfig, vm.Config{})
	state_desc := state.GetCommittedDescriptor()
	self.trie_sink.Init(&state_desc.StateRoot, opts.Trie)
	if dpos_api != nil {
		self.dpos_contract = dpos_api.NewContract(contract_storage.EVMStateStorage{&self.evm_state}, get_dpos_reader(state_desc.BlockNum), &self.evm)
	}
	if slashing_api != nil {
		self.slashing_contract = slashing_api.NewContract(contract_storage.EVMStateStorage{&self.evm_state}, get_slashing_reader(state_desc.BlockNum), &self.evm)
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
	self.evm_state.CommitTransaction(&self.trie_sink)
}

func (self *StateTransition) BlockNumber() types.BlockNum {
	return self.pending_blk_state.GetNumber()
}

func (self *StateTransition) BeginBlock(blk_info *vm.BlockInfo) {
	self.begin_block()
	blk_n := self.pending_blk_state.GetNumber()
	rules_changed := self.evm.SetBlock(&vm.Block{blk_n, *blk_info} /*self.chain_config.EVMChainConfig.Rules(blk_n)*/)
	if self.dpos_contract != nil && rules_changed {
		self.dpos_contract.Register(self.evm.RegisterPrecompiledContract)
	}
	if self.slashing_contract != nil && rules_changed {
		self.slashing_contract.Register(self.evm.RegisterPrecompiledContract)
	}
}

func (self *StateTransition) ExecuteTransaction(tx *vm.Transaction) (ret vm.ExecutionResult) {
	ret, _ = self.evm.Main(tx)
	self.evm_state_checkpoint()
	return
}

func (self *StateTransition) GetChainConfig() (ret *chain_config.ChainConfig) {
	ret = self.chain_config
	return
}

func (self *StateTransition) GetEvmState() *state_evm.EVMState {
	return &self.evm_state
}

func (self *StateTransition) DistributeRewards(rewardsStats *rewards_stats.RewardsStats, feesRewards *dpos.FeesRewards) (totalReward *uint256.Int) {
	if self.chain_config.RewardsEnabled() && rewardsStats != nil && feesRewards != nil {
		if self.dpos_contract == nil {
			panic("Stats rewards enabled but no dpos contract registered")
		}
		totalReward = self.dpos_contract.DistributeRewards(rewardsStats, feesRewards)
		self.evm_state_checkpoint()
	}

	return
}

func (self *StateTransition) EndBlock() {
	self.LastBlockNum = self.evm.GetBlock().Number
	if self.dpos_contract != nil {
		self.dpos_contract.EndBlockCall()
		self.evm_state_checkpoint()
	}
	self.pending_blk_state = nil
	return
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
		self.dpos_contract.CommitCall(self.get_dpos_reader(self.evm.GetBlock().Number))
	}
	if self.slashing_contract != nil {
		self.slashing_contract.CommitCall(self.get_slashing_reader(self.evm.GetBlock().Number))
	}
	return
}

func (self *StateTransition) AddTxFeeToBalance(account *common.Address, tx_fee *uint256.Int) {
	self.evm_state.GetAccount(account).AddBalance(tx_fee.ToBig())
}
