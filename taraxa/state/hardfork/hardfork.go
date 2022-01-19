package hardfork

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
)

type Hardforks struct {
	FixGenesisBlock uint64
}

func (h *Hardforks) IsFixGenesisFork(num types.BlockNum) bool {
	return h.FixGenesisBlock != types.BlockNumberNIL && h.FixGenesisBlock == num
}

func ApplyFixGenesisFork(balances core.BalanceMap, dpos_config *dpos.Config, state vm.State, dpos_contract *dpos.Contract) {
	dpos_contract.ResetGenesisAddresses(dpos_config.GenesisState)
	dpos_contract.UpdateConfig(*dpos_config)

	// reset genesis balances to correct state
	// we can't change balances in cpp part, so do it here
	for addr, balance := range balances {
		balance.Sub(balance, state.GetAccount(&addr).GetBalance())
		state.GetAccount(&addr).AddBalance(balance)
	}
	// Increase delegations
	dpos_contract.ApplyGenesisBalancesFixHardfork()
}
