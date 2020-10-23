package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
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
}
type ChainConfig struct {
	ETHChainConfig      params.ChainConfig
	DisableBlockRewards bool
	ExecutionOptions    vm.ExecutionOpts
	GenesisBalances     core.BalanceMap
	DPOS                *dpos.Config `rlp:"nil"`
}
type APIOpts struct {
	// TODO have single "perm-gen size" config property to derive all preallocation sizes
	ExpectedMaxTrxPerBlock        uint32
	MainTrieFullNodeLevelsToCache byte
}

func (self *API) Init(db state_db.DB, get_block_hash vm.GetHashFunc, chain_cfg ChainConfig, opts APIOpts) *API {
	self.db = db
	if chain_cfg.DPOS != nil {
		self.dpos = new(dpos.API).Init(*chain_cfg.DPOS)
	}
	self.state_transition.Init(
		self.db.GetLatestState(),
		get_block_hash,
		self.dpos,
		state_transition.ChainConfig{
			ETHChainConfig:      chain_cfg.ETHChainConfig,
			DisableBlockRewards: chain_cfg.DisableBlockRewards,
			ExecutionOptions:    chain_cfg.ExecutionOptions,
			GenesisBalances:     chain_cfg.GenesisBalances,
		},
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
	self.dry_runner.Init(self.db, get_block_hash, self.dpos, state_dry_runner.ChainConfig{
		ETHChainConfig:   chain_cfg.ETHChainConfig,
		ExecutionOptions: chain_cfg.ExecutionOptions,
	})
	return self
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
	return state_db.ExtendedReader{self.db.GetBlockState(blk_n)}
}

func (self *API) QueryDPOS(blk_n types.BlockNum) dpos.Reader {
	return self.dpos.NewReader(blk_n, func(blk_n types.BlockNum) dpos.AccountStorageReader {
		return self.ReadBlock(blk_n)
	})
}
