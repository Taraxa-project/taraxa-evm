package state

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/common"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"
)

var dpos_test_specs = []func() Spec{
	func() Spec {
		return Spec{
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
				},
				1: {
					Balances{
						addr(1): 100000000 - 1000 - 1000 - (1000 - 1),
					},
					EligibleSet{addr(1)},
				},
				2: {
					Balances{
						addr(1): 100000000 - 1000 - 1000 - (1000 - 1) - 1,
					},
					EligibleSet{addr(1)},
				},
				3: {
					Balances{
						addr(1): 100000000 - 1000 - 1000 - (1000 - 1) - 1,
					},
					EligibleSet{addr(1), addr(2)},
				},
				4: {
					Balances{
						addr(1): 100000000 - 1000 - 1000 - (1000 - 1) - 1,
					},
					EligibleSet{addr(1), addr(2), addr(3)},
				},
				6: {
					Balances{
						addr(1): 100000000 - 1000 - 1000 - (1000 - 1) - 1 + 1,
					},
					EligibleSet{addr(1), addr(3)},
				},
			},
		}
	},
	func() Spec {
		return Spec{
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
				},
				1: {
					Balances{
						addr(1): 100000000 - 1000*3 + 2,
						addr(2): 1000 - 2,
					},
					EligibleSet{addr(1), addr(2), addr(3)},
				},
				3: {
					Balances{
						addr(1): 100000000,
						addr(2): 1000,
					},
					EligibleSet{},
				},
			},
		}
	},
}

type DposTransfer = struct {
	Value    uint64
	Negative bool
}
type DposTransfers = map[common.Address]DposTransfer
type DposTransaction = struct {
	Benefactor common.Address
	DposTransfers
	ExpectedExecutionErr util.ErrorString
}
type Balances = map[common.Address]uint64
type EligibleSet = []common.Address
type ExpectedState struct {
	Balances
	EligibleSet
}
type DposGenesisState = map[common.Address]Balances
type DposCfg struct {
	EligibilityBalanceThreshold uint64
	DepositDelay                types.BlockNum
	WithdrawalDelay             types.BlockNum
	DposGenesisState
}
type GenesisBalances = map[common.Address]uint64
type DposTransactions = map[types.BlockNum][]DposTransaction
type ExpectedStates = map[types.BlockNum]ExpectedState
type Spec struct {
	GenesisBalances
	DposCfg
	DposTransactions
	ExpectedStates
}

func TestDPOS(t *testing.T) {
	for i, spec_factory := range dpos_test_specs {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			tc := tests.NewTestCtx(t)
			defer tc.Close()

			spec := spec_factory()
			var last_state_transition_blk_n types.BlockNum
			for i := range spec.DposTransactions {
				last_state_transition_blk_n = util.MaxU64(last_state_transition_blk_n, i)
			}
			last_dpos_predictable_blk_n := last_state_transition_blk_n + util.MaxU64(spec.DepositDelay, spec.WithdrawalDelay)
			asserts.Holds(len(spec.ExpectedStates) > 0)
			for i := types.BlockNum(1); i <= last_dpos_predictable_blk_n; i++ {
				if _, present := spec.ExpectedStates[i]; !present {
					spec.ExpectedStates[i] = spec.ExpectedStates[i-1]
				}
			}

			chain_cfg := base_taraxa_chain_cfg
			for k, v := range spec.GenesisBalances {
				chain_cfg.GenesisBalances[k] = new(big.Int).SetInt64(int64(v))
			}
			chain_cfg.DPOS = new(dpos.Config)
			chain_cfg.DPOS.WithdrawalDelay = spec.DposCfg.WithdrawalDelay
			chain_cfg.DPOS.DepositDelay = spec.DposCfg.DepositDelay
			chain_cfg.DPOS.EligibilityBalanceThreshold = new(big.Int).SetInt64(int64(
				spec.DposCfg.EligibilityBalanceThreshold))
			chain_cfg.DPOS.GenesisState = make(dpos.DelegatedBalanceMap)
			for k, v := range spec.DposCfg.DposGenesisState {
				v_mapped := make(dpos.BalanceMap)
				for k1, v1 := range v {
					v_mapped[k1] = new(big.Int).SetInt64(int64(v1))
				}
				chain_cfg.DPOS.GenesisState[k] = v_mapped
			}

			statedb := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
				Path: tc.DataDir(),
			})
			defer statedb.Close()
			SUT := new(API).Init(
				statedb,
				func(num types.BlockNum) *big.Int { panic("unexpected") },
				chain_cfg,
				APIOpts{},
			)
			check_exp_dpos_state := func(blk_n types.BlockNum) {
				assert_meta := "at block " + fmt.Sprint(blk_n)
				exp_st := spec.ExpectedStates[blk_n]
				dpos_reader := SUT.QueryDPOS(blk_n)
				tc.Assert.Equal(uint64(len(exp_st.EligibleSet)), dpos_reader.EligibleAddressCount(), assert_meta)
				for _, addr := range exp_st.EligibleSet {
					tc.Assert.True(dpos_reader.IsEligible(&addr), assert_meta)
				}
			}
			check_exp_state := func(blk_n types.BlockNum) {
				assert_meta := "at block " + fmt.Sprint(blk_n)
				blk_reader := SUT.ReadBlock(blk_n)
				exp_st := spec.ExpectedStates[blk_n]
				for addr, bal_expected := range exp_st.Balances {
					var bal_actual *big.Int
					blk_reader.GetAccount(&addr, func(account state_db.Account) {
						bal_actual = account.Balance
					})
					tc.Assert.True(bal_actual.IsUint64(), assert_meta)
					tc.Assert.Equal(bal_expected, bal_actual.Uint64(), assert_meta)
				}
				check_exp_dpos_state(blk_n)
				check_exp_dpos_state(blk_n + spec.DposCfg.DepositDelay)
			}
			check_exp_state(0)
			st := SUT.GetStateTransition()
			for blk_n := types.BlockNum(1); blk_n <= last_state_transition_blk_n; blk_n++ {
				st.BeginBlock(&vm.BlockInfo{})
				for _, trx := range spec.DposTransactions[blk_n] {
					dpos_addr := dpos.ContractAddress()
					res := st.ExecuteTransaction(&vm.Transaction{
						Value: bigutil.Big0,
						To:    &dpos_addr,
						From:  trx.Benefactor,
						Input: rlp.MustEncodeToBytes(trx.DposTransfers),
					})
					tc.Assert.Equal(res.ConsensusErr, util.ErrorString(""))
					tc.Assert.Equal(res.ExecutionErr, trx.ExpectedExecutionErr)
				}
				st.EndBlock(nil)
				st.Commit()
				check_exp_state(blk_n)
			}
			for i := last_state_transition_blk_n; i != 0; i-- {
				check_exp_state(i)
			}
		})
	}
}
