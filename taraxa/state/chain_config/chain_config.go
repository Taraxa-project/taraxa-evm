package chain_config

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
)

type ChainConfig struct {
	ETHChainConfig   params.ChainConfig
	ExecutionOptions vm.ExecutionOpts
	GenesisBalances  core.BalanceMap
	DPOS             *dpos.Config `rlp:"nil"`
}
