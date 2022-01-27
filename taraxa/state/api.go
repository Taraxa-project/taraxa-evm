package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_dry_runner"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
)

type API struct {
	db               state_db.DB
	state_transition state_transition.StateTransition
	dry_runner       state_dry_runner.DryRunner
	dpos             *dpos.API
	config           *chain_config.ChainConfig
}

type APIOpts struct {
	// TODO have single "perm-gen size" config property to derive all preallocation sizes
	ExpectedMaxTrxPerBlock        uint64
	MainTrieFullNodeLevelsToCache byte
}

func (self *API) Init(db state_db.DB, get_block_hash vm.GetHashFunc, chain_cfg *chain_config.ChainConfig, opts APIOpts) *API {
	self.db = db
	if chain_cfg.DPOS != nil {
		self.dpos = new(dpos.API).Init(*chain_cfg.DPOS)
	}
	self.config = chain_cfg
	self.state_transition.Init(
		self.db.GetLatestState(),
		get_block_hash,
		self.dpos,
		self.DPOSReader,
		self.config,
		state_transition.Opts{
			EVMState: state_evm.Opts{
				NumTransactionsToBuffer: opts.ExpectedMaxTrxPerBlock,
			},
			Trie: state_transition.TrieSinkOpts{
				MainTrie: trie.WriterOpts{
					FullNodeLevelsToCache: opts.MainTrieFullNodeLevelsToCache,
				},
			},
		})
	self.dry_runner.Init(self.db, get_block_hash, self.dpos, self.config)
	return self
}

func (self *API) UpdateConfig(chain_cfg *chain_config.ChainConfig) {
	self.config = chain_cfg
	self.state_transition.UpdateConfig(self.config)
	self.dry_runner.UpdateConfig(self.config)
	self.dpos.UpdateConfig(*self.config.DPOS, self.state_transition.LastBlockNum)
	// Is not updating DPOS contract config. Usually you cannot update its field without additional that processes it
	// So it should be updated separately, for example in specific hardfork function
}

func (self *API) Close() {
	self.state_transition.Close()
}

type StateTransition interface {
	BeginBlock(*vm.BlockInfo)
	ExecuteTransaction(*vm.Transaction) vm.ExecutionResult
	EndBlock([]state_common.UncleBlock)
	PrepareCommit() (state_root common.Hash)
	Commit() (state_root common.Hash)
}

func (self *API) GetStateTransition() StateTransition {
	return &self.state_transition
}

func (self *API) GetCommittedStateDescriptor() state_db.StateDescriptor {
	return self.db.GetLatestState().GetCommittedDescriptor()
}

func (self *API) DryRunTransaction(blk *vm.Block, trx *vm.Transaction, opts *vm.ExecutionOpts) vm.ExecutionResult {
	return self.dry_runner.Apply(blk, trx, opts)
}

func (self *API) ReadBlock(blk_n types.BlockNum) state_db.ExtendedReader {
	return state_db.GetBlockState(self.db, blk_n)
}

func (self *API) DPOSReader(blk_n types.BlockNum) dpos.Reader {
	// This hack is needed because deposit delay is implemented with a reader. So it is just delaying display of all changes. Because of that we can't just set different delay to immediately apply changes
	without_delay_after_hardfork := false
	if blk_n >= self.config.Hardforks.FixGenesisBlock && blk_n <= (self.config.Hardforks.FixGenesisBlock+self.config.DPOS.DepositDelay) {
		without_delay_after_hardfork = true
		// create reader with hardfork block num for 5 blocks after it to imitate delay
		blk_n = self.config.Hardforks.FixGenesisBlock
	}
	return self.dpos.NewReader(blk_n, without_delay_after_hardfork, func(blk_n types.BlockNum) dpos.StorageReader {
		return self.ReadBlock(blk_n)
	})
}
