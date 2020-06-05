package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_concurrent_schedule"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_dry_runner"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_historical"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
)

type API struct {
	Historical                   state_historical.DB
	DryRunner                    state_dry_runner.DryRunner
	ConcurrentScheduleGeneration state_concurrent_schedule.ConcurrentScheduleGeneration
	StateTransition              state_transition.StateTransition
}

func (self *API) Init(
	db state_common.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.ChainConfig,
	curr_blk_num types.BlockNum,
	curr_state_root common.Hash,
	state_transition_cache_opts state_transition.CacheOpts,
) *API {
	self.Historical = state_historical.DB{db}
	self.DryRunner.Init(self.Historical, get_block_hash, chain_cfg.EVMChainConfig)
	self.ConcurrentScheduleGeneration.Init(db, get_block_hash, chain_cfg.EVMChainConfig, curr_blk_num)
	self.StateTransition.Init(db, get_block_hash, chain_cfg, curr_blk_num, curr_state_root, state_transition_cache_opts)
	return self
}
