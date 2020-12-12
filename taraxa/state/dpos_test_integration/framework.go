package dpos_test_integration

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state"

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
type Deposits = map[common.Address]map[common.Address]DepositValue
type DepositValue struct {
	ValueNet               uint64
	ValuePendingWithdrawal uint64
}
type ExpectedState struct {
	Balances    Balances
	EligibleSet EligibleSet
	Deposits    Deposits
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

func run_specs(t *testing.T, specs []func() Spec) {
	for i, spec_factory := range specs {
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
				if s := spec.ExpectedStates[i]; s.Deposits == nil {
					s.Deposits = spec.ExpectedStates[i-1].Deposits
					spec.ExpectedStates[i] = s
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
			SUT := new(state.API).Init(
				statedb,
				func(num types.BlockNum) *big.Int { panic("unexpected") },
				chain_cfg,
				state.APIOpts{},
			)
			check_eligible_set := func(blk_n types.BlockNum) {
				assert_meta := "at block " + fmt.Sprint(blk_n)
				exp_st := spec.ExpectedStates[blk_n]
				dpos_reader := SUT.DPOSReader(blk_n)
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
				q := dpos.Query{
					AccountQueries: make(map[common.Address]dpos.AccountQuery),
				}
				acc_q := dpos.AccountQuery{
					WithOutboundDeposits: true,
					WithInboundDeposits:  true,
				}
				for benefactor, m := range exp_st.Deposits {
					q.AccountQueries[benefactor] = acc_q
					for beneficiary := range m {
						q.AccountQueries[beneficiary] = acc_q
					}
				}
				q_res := SUT.DPOSReader(blk_n).Query(&q)
				for i := 0; i < 2; i++ {
					exp_deposits := exp_st.Deposits
					if i%2 == 1 {
						exp_deposits = make(Deposits)
						for benefactor, outbound_deposits := range exp_st.Deposits {
							for beneficiary, deposit := range outbound_deposits {
								m := exp_deposits[beneficiary]
								if m == nil {
									m = make(map[common.Address]DepositValue)
									exp_deposits[beneficiary] = m
								}
								m[benefactor] = deposit
							}
						}
					}
					for addr1, deposits_by_addr1_expected := range exp_deposits {
						addr1_q_res := q_res.AccountResults[addr1]
						deposits_by_addr1_actual := addr1_q_res.OutboundDeposits
						if i%2 == 1 {
							deposits_by_addr1_actual = addr1_q_res.InboundDeposits
						}
						tc.Assert.Equal(len(deposits_by_addr1_expected), len(deposits_by_addr1_actual), assert_meta)
						for addr2, deposit_expected := range deposits_by_addr1_expected {
							deposit_actual := deposits_by_addr1_actual[addr2]
							tc.Assert.Equal(deposit_expected.ValueNet, deposit_actual.ValueNet.Uint64(), assert_meta)
							tc.Assert.Equal(
								deposit_expected.ValuePendingWithdrawal,
								deposit_actual.ValuePendingWithdrawal.Uint64(),
								assert_meta)
						}
					}
				}
				for benefactor, outbound_deposits_exp := range exp_st.Deposits {
					benefactor_q_res := q_res.AccountResults[benefactor]
					tc.Assert.Equal(len(benefactor_q_res.OutboundDeposits), len(outbound_deposits_exp), assert_meta)
					for beneficiary, deposit_expected := range outbound_deposits_exp {
						deposit_actual := benefactor_q_res.OutboundDeposits[beneficiary]
						tc.Assert.Equal(deposit_expected.ValueNet, deposit_actual.ValueNet.Uint64(), assert_meta)
						tc.Assert.Equal(
							deposit_expected.ValuePendingWithdrawal,
							deposit_actual.ValuePendingWithdrawal.Uint64(),
							assert_meta)
					}
				}
				check_eligible_set(blk_n)
				check_eligible_set(blk_n + spec.DposCfg.DepositDelay)
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
