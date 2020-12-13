package test_integration

import (
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/core/types"
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
type Deposits = map[common.Address]map[common.Address]uint64
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

func (self *Spec) run(t *testing.T) {
	tc := tests.NewTestCtx(t)
	defer tc.Close()

	var last_state_transition_blk_n types.BlockNum
	for i := range self.DposTransactions {
		last_state_transition_blk_n = util.MaxU64(last_state_transition_blk_n, i)
	}
	last_state_transition_blk_n += self.WithdrawalDelay
	last_blk_n_to_check := last_state_transition_blk_n + self.DepositDelay + 1
	asserts.Holds(len(self.ExpectedStates) > 0)
	for i := types.BlockNum(1); i <= last_blk_n_to_check; i++ {
		s_prev := self.ExpectedStates[i-1]
		s, present := self.ExpectedStates[i]
		if !present {
			s = s_prev
		}
		if s.Balances == nil {
			s.Balances = s_prev.Balances
		}
		if s.EligibleSet == nil {
			s.EligibleSet = s_prev.EligibleSet
		}
		if s.Deposits == nil {
			s.Deposits = s_prev.Deposits
		}
		self.ExpectedStates[i] = s
	}

	chain_cfg := base_taraxa_chain_cfg
	for k, v := range self.GenesisBalances {
		chain_cfg.GenesisBalances[k] = new(big.Int).SetInt64(int64(v))
	}
	chain_cfg.DPOS = new(dpos.Config)
	chain_cfg.DPOS.WithdrawalDelay = self.DposCfg.WithdrawalDelay
	chain_cfg.DPOS.DepositDelay = self.DposCfg.DepositDelay
	chain_cfg.DPOS.EligibilityBalanceThreshold = new(big.Int).SetInt64(int64(
		self.DposCfg.EligibilityBalanceThreshold))
	chain_cfg.DPOS.GenesisState = make(dpos.Addr2Addr2Balance)
	for k, v := range self.DposCfg.DposGenesisState {
		v_mapped := make(dpos.Addr2Balance)
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
	check_dpos_exp_state := func(blk_n types.BlockNum) {
		assert_meta := "at block " + fmt.Sprint(blk_n)
		exp_st := self.ExpectedStates[blk_n]
		dpos_reader := SUT.DPOSReader(blk_n)
		tc.Assert.Equal(uint64(len(exp_st.EligibleSet)), dpos_reader.EligibleAddressCount(), assert_meta)
		for _, addr := range exp_st.EligibleSet {
			tc.Assert.True(dpos_reader.IsEligible(&addr), assert_meta)
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
		q_res := dpos_reader.Query(&q)
		for i := 0; i < 2; i++ {
			exp_deposits := exp_st.Deposits
			if i%2 == 1 {
				exp_deposits = make(Deposits)
				for benefactor, outbound_deposits := range exp_st.Deposits {
					for beneficiary, deposit := range outbound_deposits {
						m := exp_deposits[beneficiary]
						if m == nil {
							m = make(map[common.Address]uint64)
							exp_deposits[beneficiary] = m
						}
						m[benefactor] = deposit
					}
				}
			}
			for addr1, deposits_by_addr1_expected := range exp_deposits {
				addr1_q_res := q_res.AccountResults[addr1]
				deposits_by_addr1_actual := addr1_q_res.OutboundDeposits
				assert_meta_ := assert_meta + " (out)"
				if i%2 == 1 {
					deposits_by_addr1_actual = addr1_q_res.InboundDeposits
					assert_meta_ = assert_meta + " (in)"
				}
				assert_meta_ += " @ " + addr1.Hex()
				tc.Assert.Equal(len(deposits_by_addr1_expected), len(deposits_by_addr1_actual), assert_meta_)
				for addr2, deposit_expected := range deposits_by_addr1_expected {
					assert_meta__ := assert_meta_ + " -> " + addr2.Hex()
					deposit_actual := deposits_by_addr1_actual[addr2]
					tc.Assert.Equal(deposit_expected, deposit_actual.Uint64(), assert_meta__)
				}
			}
		}
	}
	check_exp_state := func(blk_n types.BlockNum) {
		assert_meta := "at block " + fmt.Sprint(blk_n)
		blk_reader := SUT.ReadBlock(blk_n)
		exp_st := self.ExpectedStates[blk_n]
		for addr, bal_expected := range exp_st.Balances {
			var bal_actual *big.Int
			blk_reader.GetAccount(&addr, func(account state_db.Account) {
				bal_actual = account.Balance
			})
			tc.Assert.True(bal_actual.IsUint64(), assert_meta)
			tc.Assert.Equal(bal_expected, bal_actual.Uint64(), assert_meta)
		}
		check_dpos_exp_state(blk_n)
		check_dpos_exp_state(blk_n + self.DposCfg.DepositDelay)
	}
	check_exp_state(0)
	st := SUT.GetStateTransition()
	for blk_n := types.BlockNum(1); blk_n <= last_state_transition_blk_n; blk_n++ {
		st.BeginBlock(&vm.BlockInfo{})
		for _, trx := range self.DposTransactions[blk_n] {
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
}

type Specs []Spec

func (self *Specs) add(spec Spec) int {
	*self = append(*self, spec)
	return 0
}

func (self *Specs) add_factory(spec func() Spec) int {
	return self.add(spec())
}

func (self *Specs) run(t *testing.T) {
	for i, spec := range *self {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			spec.run(t)
		})
	}
}
