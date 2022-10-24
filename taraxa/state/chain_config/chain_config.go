package chain_config

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/params"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
)

type BlockRewardsOpts struct {
	// Disables new tokens generation as block reward
	DisableBlockRewards bool

	// TODO: once we fix tests, this flag can be deleted as rewards should be processed only in dpos contract
	// Disbales rewards distribution through contract - rewards are added directly to the validators accounts
	DisableContractDistribution bool
}

type ChainConfig struct {
	ETHChainConfig      params.ChainConfig
	BlockRewardsOptions BlockRewardsOpts
	GenesisBalances     core.BalanceMap
	DPOS                dpos.Config
}
