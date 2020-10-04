package state

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_dry_runner"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
)

type API struct {
	db               state_db.DB
	state_transition state_transition.StateTransition
	dry_runner       state_dry_runner.DryRunner
	dpos             *dpos.API
}
type ChainConfig struct {
	ExecutionConfig state_common.ExecutionConfig
	GenesisBalances core.BalanceMap
	DPOS            *dpos.Config
}
type Opts struct {
	StateTransition state_transition.Opts
}

func (self *API) Init(db state_db.DB, get_block_hash vm.GetHashFunc, chain_cfg ChainConfig, opts Opts) *API {
	if chain_cfg.DPOS != nil {
		self.dpos = new(dpos.API).Init(*chain_cfg.DPOS)
	}
	self.state_transition.Init(
		db.GetLatestState(),
		get_block_hash,
		self.dpos,
		chain_cfg.ExecutionConfig,
		chain_cfg.GenesisBalances,
		opts.StateTransition)
	self.dry_runner.Init(db, get_block_hash, self.dpos, chain_cfg.ExecutionConfig)
	return self
}

func (self *API) ReadBlock(blk_n types.BlockNum) state_db.ExtendedReader {
	return state_db.ExtendedReader{self.db.GetBlockState(blk_n)}
}
