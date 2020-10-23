package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"
)

var base_taraxa_chain_cfg = ChainConfig{
	DisableBlockRewards: true,
	ExecutionOptions: vm.ExecutionOpts{
		DisableGasFee:     true,
		DisableNonceCheck: true,
	},
	ETHChainConfig: params.ChainConfig{
		DAOForkBlock: types.BlockNumberNIL,
	},
	GenesisBalances: make(core.BalanceMap),
}

var addr = tests.SimpleAddr

func addr_p(i int) *common.Address {
	ret := addr(i)
	return &ret
}
