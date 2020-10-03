package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_dry_runner"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
)

type API struct {
	DryRunner       state_dry_runner.DryRunner
	StateTransition state_transition.StateTransition
}

func (self *API) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.ChainConfig,
	curr_blk_num types.BlockNum,
	curr_state_root *common.Hash,
	state_transition_cache_opts state_transition.StateTransitionOpts,
) *API {
	self.DryRunner.Init(db, get_block_hash, chain_cfg.Execution)
	self.StateTransition.Init(db, get_block_hash, chain_cfg, curr_blk_num, curr_state_root, state_transition_cache_opts)
	return self
}
