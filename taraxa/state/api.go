package state

import (
	"math/big"
	"sort"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	dpos_2 "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos_2.0/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_dry_runner"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
)

type API struct {
	rocksdb          *state_db_rocksdb.DB
	db               state_db.DB
	state_transition state_transition.StateTransition
	dry_runner       state_dry_runner.DryRunner
	dpos             *dpos.API
	dpos2            *dpos_2.API
	config           *chain_config.ChainConfig
}

type APIOpts struct {
	// TODO have single "perm-gen size" config property to derive all preallocation sizes
	ExpectedMaxTrxPerBlock        uint64
	MainTrieFullNodeLevelsToCache byte
}

func (self *API) Init(db *state_db_rocksdb.DB, get_block_hash vm.GetHashFunc, chain_cfg *chain_config.ChainConfig, opts APIOpts) *API {
	self.db = db
	self.rocksdb = db
	self.config = chain_cfg

	if self.config.DPOS != nil {
		self.dpos = new(dpos.API).Init(*self.config.DPOS)
		config_changes := self.rocksdb.GetDPOSConfigChanges()
		if len(config_changes) == 0 {

			bytes := rlp.MustEncodeToBytes(*self.config.DPOS)
			self.rocksdb.SaveDPOSConfigChange(0, bytes)
			self.dpos.UpdateConfig(0, *self.config.DPOS)
		} else {
			// Order mapping keys to apply changes in correct order
			keys := make([]uint64, 0)
			for k, _ := range config_changes {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

			// Decode rlp data from db and apply
			for _, key := range keys {
				value := config_changes[key]
				cfg := new(dpos.Config)
				rlp.MustDecodeBytes(value, cfg)
				self.dpos.UpdateConfig(key, *cfg)
			}
		}
	}

	if self.config.DPOS != nil {
		self.dpos2 = new(dpos_2.API).Init(*self.config.DPOS)
		config_changes := self.rocksdb.GetDPOSConfigChanges()
		if len(config_changes) == 0 {
			self.dpos2.UpdateConfig(0, *self.config.DPOS)
		} else {
			// Order mapping keys to apply changes in correct order
			keys := make([]uint64, 0)
			for k, _ := range config_changes {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

			// Decode rlp data from db and apply
			for _, key := range keys {
				value := config_changes[key]
				cfg := new(dpos.Config)
				rlp.MustDecodeBytes(value, cfg)
				self.dpos2.UpdateConfig(key, *cfg)
			}
		}
	}


	self.state_transition.Init(
		self.db.GetLatestState(),
		get_block_hash,
		self.dpos,
		self.dpos2,
		self.DPOS2Reader,
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
	self.dry_runner.Init(self.db, get_block_hash, self.dpos, self.dpos2, self.DPOS2Reader, self.config)
	return self
}

func (self *API) UpdateConfig(chain_cfg *chain_config.ChainConfig) {
	self.config = chain_cfg
	self.state_transition.UpdateConfig(self.config)
	self.dry_runner.UpdateConfig(self.config)
	config_update_block_num := self.state_transition.LastBlockNum + 1
	self.dpos.UpdateConfig(config_update_block_num, *self.config.DPOS)
	self.rocksdb.SaveDPOSConfigChange(config_update_block_num, rlp.MustEncodeToBytes(self.config.DPOS))
	// Is not updating DPOS contract config. Usually you cannot update its field without additional that processes it
	// So it should be updated separately, for example in specific hardfork function
}

func (self *API) Close() {
	self.state_transition.Close()
}

type StateTransition interface {
	BeginBlock(*vm.BlockInfo, map[common.Address]*big.Int)
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

func (self *API) DPOS2Reader(blk_n types.BlockNum) dpos_2.Reader {
	if blk_n >= self.config.Hardforks.FixGenesisBlock && blk_n <= (self.config.Hardforks.FixGenesisBlock+self.config.DPOS.DepositDelay) {
	// create reader with hardfork block num for 5 blocks after it to imitate delay
		blk_n = self.config.Hardforks.FixGenesisBlock
	}
	return self.dpos2.NewReader(blk_n, func(blk_n types.BlockNum) dpos_2.StorageReader {
		return self.ReadBlock(blk_n)
	})
}
