package test_integration

import (
	"math/big"
	"strings"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	dpos_2 "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos_2.0/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
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
	Balances         Balances
	EligibleSet      EligibleSet
	OutboundDeposits Deposits
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


var addr, addr_p = tests.Addr, tests.AddrP

type DposTest struct {
	GenesisBalances
	DposCfg
	st        state.StateTransition
	statedb   *state_db_rocksdb.DB
	tc        *tests.TestCtx
	SUT       *state.API
	blk_n     types.BlockNum
	dpos_addr common.Address
	abi       abi.ABI
}

var base_taraxa_chain_cfg = chain_config.ChainConfig{
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

func (self *DposTest) init(t *tests.TestCtx) {
	self.tc = t

	chain_cfg := base_taraxa_chain_cfg
	for k, v := range self.GenesisBalances {
		chain_cfg.GenesisBalances[k] = new(big.Int).SetInt64(int64(v))
	}
	chain_cfg.DPOS = new(dpos.Config)
	chain_cfg.DPOS.WithdrawalDelay = self.DposCfg.WithdrawalDelay
	chain_cfg.DPOS.DepositDelay = self.DposCfg.DepositDelay
	chain_cfg.DPOS.EligibilityBalanceThreshold = new(big.Int).SetInt64(int64(
		self.DposCfg.EligibilityBalanceThreshold))
	chain_cfg.DPOS.VoteEligibilityBalanceStep = new(big.Int).SetInt64(int64(
		self.DposCfg.EligibilityBalanceThreshold))

	for k, v := range self.DposCfg.DposGenesisState {
		entry := dpos.GenesisStateEntry{Benefactor: k}
		for k1, v1 := range v {
			entry.Transfers = append(entry.Transfers, dpos.GenesisTransfer{
				Beneficiary: k1,
				Value:       new(big.Int).SetInt64(int64(v1)),
			})
		}
		chain_cfg.DPOS.GenesisState = append(chain_cfg.DPOS.GenesisState, entry)
	}

	self.statedb = new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: self.tc.DataDir(),
	})
	self.SUT = new(state.API).Init(
		self.statedb,
		func(num types.BlockNum) *big.Int { panic("unexpected") },
		&chain_cfg,
		state.APIOpts{},
	)

	self.st = self.SUT.GetStateTransition()
	self.dpos_addr = dpos_2.ContractAddress()
	self.abi, _ = abi.JSON(strings.NewReader(dpos_2.TaraxaDposClientMetaData))
}

func (self *DposTest) execute(from common.Address, value uint64, input []byte) vm.ExecutionResult {
	self.blk_n++
	self.st.BeginBlock(&vm.BlockInfo{}, nil)

	res := self.st.ExecuteTransaction(&vm.Transaction{
		Value: new(big.Int).SetUint64(value),
		To:    &self.dpos_addr,
		From:  from,
		Input: input,
	})

	self.st.EndBlock(nil)
	self.st.Commit()
	return res
}

func (self *DposTest) AddRewards(rewards map[common.Address]*big.Int)  {
	self.blk_n++
	self.st.BeginBlock(&vm.BlockInfo{}, rewards)

	res := self.st.ExecuteTransaction(&vm.Transaction{
		Value: new(big.Int).SetUint64(0),
		To:    &self.dpos_addr,
		From:  addr(1),
		Input: []byte("0x123213"),
	})

	self.st.EndBlock(nil)
	self.st.Commit()
	self.tc.Assert.Equal(util.ErrorString(""), res.ConsensusErr)
	self.tc.Assert.Equal(util.ErrorString(""), res.ExecutionErr)
}

func (self *DposTest) GetBalance(account common.Address) (*big.Int) {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	return bal_actual
}

func (self *DposTest) ExecuteAndCheck(from common.Address, value uint64, input []byte, exe_err util.ErrorString, cons_err util.ErrorString)  {
	res := self.execute(from, value, input)
	self.tc.Assert.Equal(cons_err, res.ConsensusErr)
	self.tc.Assert.Equal(exe_err, res.ExecutionErr)
}

func (self *DposTest) end() {
	self.statedb.Close()
	self.tc.Close()
}

func (self *DposTest) pack(name string, args ...interface{}) []byte {
	packed, err := self.abi.Pack(name, args...)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return packed
}


func init_config(t *testing.T) (tc tests.TestCtx, test DposTest){
	tc = tests.NewTestCtx(t)
	test.GenesisBalances = GenesisBalances{addr(1): 100000000, addr(2): 100000000, addr(3): 100000000}
	test.DposCfg = DposCfg{
		EligibilityBalanceThreshold: 1000,
		DepositDelay:                2,
		WithdrawalDelay:             4,
		DposGenesisState: DposGenesisState{
			addr(1): {
				addr(1): 1000,
			},
		},
	}
	test.init(&tc)
	return
}