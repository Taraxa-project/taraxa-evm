package state_config

import (
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
)

type ChainConfig struct {
	Execution ExecutionConfig
	DPOS      *dpos.Config
}
type ExecutionConfig struct {
	DisableBlockRewards bool
	ETHForks            params.ChainConfig
	Options             vm.ExecutionOptions
}
