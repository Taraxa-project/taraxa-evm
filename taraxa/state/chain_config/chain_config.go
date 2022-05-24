package chain_config

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
)

type ChainConfig struct {
	ETHChainConfig      params.ChainConfig
	DisableBlockRewards bool
	ExecutionOptions    vm.ExecutionOpts
	GenesisBalances     core.BalanceMap
	DPOS                *dpos.Config `rlp:"nil"`
}
