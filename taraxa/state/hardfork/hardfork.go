package hardfork

import (
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
)

type Hardforks struct {
	FixGenesisBlockNum uint64
}

func (h *Hardforks) ApplyFixGenesisHardfork(balances core.BalanceMap, dpos_cfg *dpos.Config, state vm.State, dpos_contract *dpos.Contract, dpos_reader dpos.Reader) {
	// multiply genesis balances
	// we can't change balances in cpp part, so do it here
	for addr, balance := range balances {
		balance.Sub(balance, state.GetAccount(&addr).GetBalance())
		state.GetAccount(&addr).AddBalance(balance)
	}
	// Increase delegations
	dpos_contract.ApplyGenesisBalancesFixHardfork()
}

func (h *Hardforks) IsFixGenesisFork(num types.BlockNum) bool {
	return h.FixGenesisBlockNum != types.BlockNumberNIL && h.FixGenesisBlockNum == num
}
