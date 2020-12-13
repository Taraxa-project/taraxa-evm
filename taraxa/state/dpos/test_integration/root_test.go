package test_integration

import (
	"testing"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
)

var specs Specs
var _ = specs.add(Spec{
	GenesisBalances{
		addr(1): 100000000,
	},
	DposCfg{
		EligibilityBalanceThreshold: 1000,
		DepositDelay:                2,
		WithdrawalDelay:             4,
		DposGenesisState: DposGenesisState{
			addr(1): {
				addr(1): 1000,
			},
		},
	},
	DposTransactions{
		1: {
			{
				Benefactor: addr(1),
				DposTransfers: DposTransfers{
					addr(2): {Value: 1000},
					addr(3): {Value: 1000 - 1},
				},
			},
		},
		2: {
			{
				Benefactor: addr(1),
				DposTransfers: DposTransfers{
					addr(2): {Value: 1000, Negative: true},
					addr(3): {Value: 1},
				},
			},
		},
	},
	ExpectedStates{
		0: {
			Balances{
				addr(1): 100000000 - 1000,
			},
			EligibleSet{addr(1)},
			Deposits{
				addr(1): {
					addr(1): 1000,
				},
			},
		},
		1: {
			Balances{
				addr(1): 100000000 - 1000 - 1000 - (1000 - 1),
			},
			nil,
			nil,
		},
		2: {
			Balances{
				addr(1): 100000000 - 1000 - 1000 - (1000 - 1) - 1,
			},
			nil,
			nil,
		},
		3: {
			nil,
			EligibleSet{addr(1), addr(2)},
			Deposits{
				addr(1): {
					addr(1): 1000,
					addr(2): 1000,
					addr(3): 1000 - 1,
				},
			},
		},
		4: {
			nil,
			EligibleSet{addr(1), addr(2), addr(3)},
			Deposits{
				addr(1): {
					addr(1): 1000,
					addr(2): 1000,
					addr(3): 1000 - 1 + 1,
				},
			},
		},
		6: {
			Balances{
				addr(1): 100000000 - 1000 - 1000 - (1000 - 1) - 1 + 1000,
			},
			EligibleSet{addr(1), addr(3)},
			Deposits{
				addr(1): {
					addr(1): 1000,
					addr(3): 1000 - 1 + 1,
				},
			},
		},
	},
})

var _ = specs.add(Spec{
	GenesisBalances{
		addr(1): 100000000,
		addr(2): 1000,
	},
	DposCfg{
		EligibilityBalanceThreshold: 1000,
		DepositDelay:                0,
		WithdrawalDelay:             0,
		DposGenesisState: DposGenesisState{
			addr(1): {
				addr(1): 1000,
				addr(2): 1000,
				addr(3): 1000,
			},
		},
	},
	DposTransactions{
		1: {
			{
				Benefactor: addr(1),
				DposTransfers: DposTransfers{
					addr(2): {Value: 1, Negative: true},
					addr(3): {Value: 1, Negative: true},
				},
			},
			{
				Benefactor: addr(2),
				DposTransfers: DposTransfers{
					addr(2): {Value: 1},
					addr(3): {Value: 1},
				},
			},
		},
		2: {
			{
				Benefactor: addr(3),
				DposTransfers: DposTransfers{
					addr(2): {Value: 33, Negative: true},
					addr(3): {Value: 1, Negative: true},
				},
				ExpectedExecutionErr: dpos.ErrWithdrawalExceedsDeposit,
			},
		},
		3: {
			{
				Benefactor: addr(1),
				DposTransfers: DposTransfers{
					addr(1): {Value: 1000, Negative: true},
					addr(2): {Value: 1000 - 1, Negative: true},
					addr(3): {Value: 1000 - 1, Negative: true},
				},
			},
			{
				Benefactor: addr(2),
				DposTransfers: DposTransfers{
					addr(2): {Value: 1, Negative: true},
					addr(3): {Value: 1, Negative: true},
				},
			},
		},
	},
	ExpectedStates{
		0: {
			Balances{
				addr(1): 100000000 - 1000*3,
			},
			EligibleSet{addr(1), addr(2), addr(3)},
			Deposits{
				addr(1): {
					addr(1): 1000,
					addr(2): 1000,
					addr(3): 1000,
				},
			},
		},
		1: {
			Balances{
				addr(1): 100000000 - 1000*3 + 2,
				addr(2): 1000 - 2,
			},
			EligibleSet{addr(1), addr(2), addr(3)},
			Deposits{
				addr(1): {
					addr(1): 1000,
					addr(2): 999,
					addr(3): 999,
				},
				addr(2): {
					addr(2): 1,
					addr(3): 1,
				},
			},
		},
		3: {
			Balances{
				addr(1): 100000000,
				addr(2): 1000,
			},
			EligibleSet{},
			Deposits{
				addr(1): {},
				addr(2): {},
			},
		},
	},
})

func TestRoot(t *testing.T) {
	specs.run(t)
}
