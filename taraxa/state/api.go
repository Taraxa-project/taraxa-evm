package state

import (
	"sort"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_dry_runner"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/holiman/uint256"
)

type API struct {
	rocksdb          *state_db_rocksdb.DB
	db               state_db.DB
	state_transition state_transition.StateTransition
	dry_runner       state_dry_runner.DryRunner
	trace_runner     state_dry_runner.TraceRunner
	dpos             *dpos.API
	slashing         *slashing.API
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

	self.dpos = new(dpos.API).Init(*self.config)
	self.slashing = new(slashing.API).Init(self.config.DPOS.Slashing)
	config_changes := self.rocksdb.GetDPOSConfigChanges()
	if len(config_changes) == 0 {
		bytes := rlp.MustEncodeToBytes(self.config.DPOS)
		self.rocksdb.SaveDPOSConfigChange(0, bytes)
		self.dpos.UpdateConfig(0, *self.config)
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
			cfg := new(chain_config.DposConfig)
			rlp.MustDecodeBytes(value, cfg)
			self.dpos.UpdateConfig(key, *cfg)
		}
	}

	self.state_transition.Init(
		self.db.GetLatestState(),
		get_block_hash,
		self.dpos,
		self.DPOSReader,
		self.slashing,
		self.SlashingReader,
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
	self.dry_runner.Init(self.db, get_block_hash, self.dpos, self.DPOSReader, self.slashing, self.SlashingReader, self.config)
	self.trace_runner.Init(self.db, get_block_hash, self.dpos, self.DPOSReader, self.slashing, self.SlashingReader, self.config)
	return self
}

func (self *API) UpdateConfig(chain_cfg *chain_config.ChainConfig) {
	self.config = chain_cfg
	self.state_transition.UpdateConfig(self.config)
	self.dry_runner.UpdateConfig(self.config)
	self.trace_runner.UpdateConfig(self.config)
	config_update_block_num := self.state_transition.LastBlockNum + 1
	self.dpos.UpdateConfig(config_update_block_num, *self.config)
	self.rocksdb.SaveDPOSConfigChange(config_update_block_num, rlp.MustEncodeToBytes(self.config.DPOS))
	// Is not updating DPOS contract config. Usually you cannot update its field without additional that processes it
	// So it should be updated separately, for example in specific hardfork function
}

func (self *API) Close() {
	self.state_transition.Close()
}

type StateTransition interface {
	BeginBlock(*vm.BlockInfo)
	BlockNumber() types.BlockNum
	ExecuteTransaction(*vm.Transaction) vm.ExecutionResult
	AddTxFeeToBalance(account *common.Address, tx_fee *uint256.Int)
	GetChainConfig() *chain_config.ChainConfig
	GetEvmState() *state_evm.EVMState
	DistributeRewards(*rewards_stats.RewardsStats) *uint256.Int
	EndBlock()
	PrepareCommit() (state_root common.Hash)
	Commit() (state_root common.Hash)
}

func (self *API) GetStateTransition() StateTransition {
	return &self.state_transition
}

func (self *API) GetCommittedStateDescriptor() state_db.StateDescriptor {
	return self.db.GetLatestState().GetCommittedDescriptor()
}

func (self *API) DryRunTransaction(blk *vm.Block, trx *vm.Transaction) vm.ExecutionResult {
	return self.dry_runner.Apply(blk, trx)
}

func (self *API) Trace(blk *vm.Block, trxs *[]vm.Transaction, conf *vm.TracingConfig) []byte {
	return self.trace_runner.Trace(blk, trxs, conf)
}

func (self *API) ReadBlock(blk_n types.BlockNum) state_db.ExtendedReader {
	return state_db.GetBlockState(self.db, blk_n)
}

func (self *API) DPOSReader(blk_n types.BlockNum) dpos.Reader {
	return self.dpos.NewReader(blk_n, func(blk_n types.BlockNum) contract_storage.StorageReader {
		return self.ReadBlock(blk_n)
	})
}

func (self *API) SlashingReader(blk_n types.BlockNum) slashing.Reader {
	return self.slashing.NewReader(blk_n, func(blk_n types.BlockNum) contract_storage.StorageReader {
		return self.ReadBlock(blk_n)
	})
}
