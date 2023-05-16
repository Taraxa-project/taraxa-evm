package chain_config

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/params"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
)

type Hardforks struct {
	FixRedelegateBlockNum uint64
}

type ChainConfig struct {
	EVMChainConfig  params.ChainConfig
	GenesisBalances core.BalanceMap
	DPOS            dpos.Config
	Hardforks       Hardforks
}

func (self *ChainConfig) RewardsEnabled() bool {
	return self.DPOS.YieldPercentage > 0
}
