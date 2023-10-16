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

func (st *StateTransition) Init(
	state state_db.LatestState,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	get_dpos_reader func(types.BlockNum) dpos.Reader,
	slashing_api *slashing.API,
	get_slashing_reader func(types.BlockNum) slashing.Reader,
	chain_config *chain_config.ChainConfig,
	opts Opts,
) *StateTransition {
	st.chain_config = chain_config
	st.state = state
	st.evm_state.Init(opts.EVMState)
	st.get_dpos_reader = get_dpos_reader
	st.get_slashing_reader = get_slashing_reader
	st.evm.Init(get_block_hash, &st.evm_state, vm.Opts{
		// 24MB total
		PreallocatedMem: 8 * 1024 * 1024,
	}, st.chain_config.EVMChainConfig, vm.Config{})
	state_desc := state.GetCommittedDescriptor()
	st.trie_sink.Init(&state_desc.StateRoot, opts.Trie)
	if dpos_api != nil {
		st.dpos_contract = dpos_api.NewContract(contract_storage.EVMStateStorage{EVMState: &st.evm_state}, get_dpos_reader(state_desc.BlockNum), &st.evm)
	}
	if slashing_api != nil {
		st.slashing_contract = slashing_api.NewContract(contract_storage.EVMStateStorage{EVMState: &st.evm_state}, get_slashing_reader(state_desc.BlockNum), &st.evm)
	}
	if state_common.IsEmptyStateRoot(&state_desc.StateRoot) {
		st.begin_block()
		asserts.Holds(st.pending_blk_state.GetNumber() == 0)
		for addr, balance := range st.chain_config.GenesisBalances {
			st.evm_state.GetAccount(&addr).AddBalance(balance)
		}
		if st.dpos_contract != nil {
			util.PanicIfNotNil(st.dpos_contract.ApplyGenesis(st.evm_state.GetAccount))
		}
		st.evm_state_checkpoint()
		st.Commit()
	}
	// we need genesis balances later, so it is commented
	// st.chain_config.GenesisBalances = nil
	return st
}

func (st *StateTransition) UpdateConfig(cfg *chain_config.ChainConfig) {
	st.new_chain_config = cfg
}

func (st *StateTransition) Close() {
	st.trie_sink.Close()
}

func (st *StateTransition) begin_block() {
	st.pending_blk_state = st.state.BeginPendingBlock()
	st.evm_state.SetInput(state_db.ExtendedReader{Reader: st.pending_blk_state})
	st.trie_sink.SetIO(st.pending_blk_state)
}

func (st *StateTransition) evm_state_checkpoint() {
	st.evm_state.CommitTransaction(&st.trie_sink)
}

func (st *StateTransition) BlockNumber() types.BlockNum {
	return st.pending_blk_state.GetNumber()
}

func (st *StateTransition) BeginBlock(blk_info *vm.BlockInfo) {
	st.begin_block()
	blk_n := st.pending_blk_state.GetNumber()
	rules_changed := st.evm.SetBlock(&vm.Block{Number: blk_n, BlockInfo: *blk_info} /*st.chain_config.EVMChainConfig.Rules(blk_n)*/)
	if st.dpos_contract != nil && rules_changed {
		st.dpos_contract.Register(st.evm.RegisterPrecompiledContract)
	}
	if st.slashing_contract != nil && st.chain_config.Hardforks.IsMagnoliaHardfork(blk_n) && rules_changed {
		st.slashing_contract.Register(st.evm.RegisterPrecompiledContract)
	}
}

func (st *StateTransition) ExecuteTransaction(tx *vm.Transaction) (ret vm.ExecutionResult) {
	ret, _ = st.evm.Main(tx)
	st.evm_state_checkpoint()
	return
}

func (st *StateTransition) GetChainConfig() (ret *chain_config.ChainConfig) {
	ret = st.chain_config
	return
}

func (st *StateTransition) GetEvmState() *state_evm.EVMState {
	return &st.evm_state
}

func (st *StateTransition) DistributeRewards(rewardsStats *rewards_stats.RewardsStats) (totalReward *uint256.Int) {
	if st.chain_config.RewardsEnabled() && rewardsStats != nil {
		if st.dpos_contract == nil {
			panic("Stats rewards enabled but no dpos contract registered")
		}
		totalReward = st.dpos_contract.DistributeRewards(rewardsStats)
		st.evm_state_checkpoint()
	}

	return
}

func (st *StateTransition) EndBlock() {
	st.LastBlockNum = st.evm.GetBlock().Number
	if st.dpos_contract != nil {
		st.dpos_contract.EndBlockCall(st.LastBlockNum)
		st.slashing_contract.CleanupJailedValidators(st.LastBlockNum)
		st.evm_state_checkpoint()
	}
	st.pending_blk_state = nil
}

func (st *StateTransition) PrepareCommit() common.Hash {
	st.evm_state.Commit()
	st.evm_state.SetInput(nil)
	st.pending_state_root = st.trie_sink.Commit()
	st.trie_sink.SetIO(nil)
	return st.pending_state_root
}

func (st *StateTransition) Commit() (state_root common.Hash) {
	if st.pending_state_root == common.ZeroHash {
		st.PrepareCommit()
	}
	state_root, st.pending_state_root = st.pending_state_root, common.ZeroHash
	util.PanicIfNotNil(st.state.Commit(state_root)) // TODO move out of here, this should be async
	if st.dpos_contract != nil {
		st.dpos_contract.CommitCall(st.get_dpos_reader(st.evm.GetBlock().Number))
	}
	if st.slashing_contract != nil && st.chain_config.Hardforks.IsMagnoliaHardfork(st.evm.GetBlock().Number) {
		st.slashing_contract.CommitCall(st.get_slashing_reader(st.evm.GetBlock().Number))
	}
	return
}

func (st *StateTransition) AddTxFeeToBalance(account *common.Address, tx_fee *uint256.Int) {
	st.evm_state.GetAccount(account).AddBalance(tx_fee.ToBig())
}

// TODO: remove after implementing proper test
func (st *StateTransition) GetJailedValidators() (jailed_validators []common.Address) {
	reader := st.get_slashing_reader(st.evm.GetBlock().Number)
	return reader.GetJailedValidators()
}
