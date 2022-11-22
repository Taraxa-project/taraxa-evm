package chain_config

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/params"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
)

type ChainConfig struct {
	ETHChainConfig  params.ChainConfig
	GenesisBalances core.BalanceMap
	DPOS            dpos.Config
}

func (self *ChainConfig) RewardsEnabled() bool {
	return self.DPOS.YieldPercentage > 0
}
