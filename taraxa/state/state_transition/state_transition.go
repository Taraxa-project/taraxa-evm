package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"

	"github.com/Taraxa-project/taraxa-evm/core"

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
	db                state_db.DB
	dpos_contract     *dpos.Contract
	curr_blk_num      types.BlockNum
	evm               vm.EVM
	last_block_reader state_db.BlockReader
	evm_state         state_evm.EVMState
	trie_sink         TrieSink
}

type StateTransitionOpts struct {
	TrieWriters               TrieWriterOpts
	ExpectedMaxNumTrxPerBlock uint32
}

func (self *StateTransition) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.ChainConfig,
	curr_blk_num types.BlockNum, // TODO use types.BlockNumberNIL instead of 0
	curr_state_root *common.Hash,
	opts StateTransitionOpts,
) *StateTransition {
	self.db = db
	self.exec_cfg = chain_cfg.Execution
	self.curr_blk_num = curr_blk_num
	self.last_block_reader = state_db.BlockReader{db.ReadBlock(self.curr_blk_num)}
	defer self.last_block_reader.NotifyDone()
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
	return self.Commit()
}

func (self *StateTransition) begin_block() {
	db_tx := self.db.WriteBlock(self.curr_blk_num)
	self.last_block_reader.Tx = db_tx
	self.trie_sink.SetTransaction(db_tx)
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

func (self *StateTransition) ExecuteTransaction(trx *vm.Transaction) (ret vm.ExecutionResult) {
	ret = self.evm.Main(trx, self.exec_cfg.Options)
	self.evm_state_checkpoint()
	return
}

func (self *StateTransition) EndBlock(uncles []state_common.UncleBlock) {
	defer self.last_block_reader.NotifyDone()
	if self.dpos_contract != nil {
		self.dpos_contract.Commit(self.curr_blk_num)
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

func (self *StateTransition) Commit() *common.Hash {
	self.evm_state.Commit()
	return self.trie_sink.Commit()
}
