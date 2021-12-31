package hardfork

import (
	"fmt"
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
)

type Hardforks struct {
	FixGenesisBlockNum uint64
}

func (h *Hardforks) ApplyFixGenesisHardfork(balances core.BalanceMap, dpos_cfg *dpos.Config, state vm.State, dpos_contract *dpos.Contract) {
	// multiply genesis balances
	// we can't change balances in cpp part, so do it here
	fmt.Println("APPLY GO PART OF HARDFORK")
	mul_power := big.NewInt(1e+18)
	for addr, balance := range balances {
		balance_to_add := big.NewInt(0)
		balance_to_add.Mul(balance, mul_power).Sub(balance_to_add, balance)
		state.GetAccount(&addr).AddBalance(balance_to_add)
	}
	// Increase delegations?
	for _, entry := range dpos_cfg.GenesisState {
		transfers := make([]dpos.BeneficiaryAndTransfer, len(entry.Transfers))
		for i, v := range entry.Transfers {
			transfers[i] = dpos.BeneficiaryAndTransfer{
				Beneficiary: v.Beneficiary,
				Transfer:    dpos.Transfer{Value: v.Value},
			}
		}
		if err := dpos_contract.ApplyTransfers(entry.Benefactor, transfers); err != nil {
			panic(err)
		}
	}
}

func (h *Hardforks) IsFixGenesisFork(num types.BlockNum) bool {
	return h.FixGenesisBlockNum != types.BlockNumberNIL && h.FixGenesisBlockNum == num
}
