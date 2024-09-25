package test_utils

import (
	"math/big"
	"strings"
	"testing"

	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
)

type ContractTest struct {
	Chain_cfg     chain_config.ChainConfig
	St            state.StateTransition
	contract_addr common.Address
	Statedb       *state_db_rocksdb.DB
	tc            *tests.TestCtx
	SUT           *state.API
	blk_n         types.BlockNum
	slashing_addr common.Address
	abi           abi.ABI
}

func Init_test(contract_addr *common.Address, contract_abi string, t *testing.T, cfg chain_config.ChainConfig) (tc tests.TestCtx, test ContractTest) {
	tc = tests.NewTestCtx(t)
	test.init(*contract_addr, contract_abi, &tc, cfg)
	return
}

func (self *ContractTest) init(contract_addr common.Address, contract_abi string, t *tests.TestCtx, cfg chain_config.ChainConfig) {
	self.tc = t
	self.Chain_cfg = cfg

	self.Statedb = new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: self.tc.DataDir(),
	})
	self.SUT = new(state.API).Init(
		self.Statedb,
		func(num types.BlockNum) *big.Int { panic("unexpected") },
		&self.Chain_cfg,
		state.APIOpts{},
	)

	self.St = self.SUT.GetStateTransition()
	self.contract_addr = contract_addr
	self.abi, _ = abi.JSON(strings.NewReader(contract_abi))
}

func (self *ContractTest) execute(from common.Address, value *big.Int, input []byte) vm.ExecutionResult {
	senderNonce := self.GetNonce(from)
	senderNonce.Add(senderNonce, big.NewInt(1))

	self.blk_n++
	self.St.BeginBlock(&vm.BlockInfo{})

	res := self.St.ExecuteTransaction(&vm.Transaction{
		Value:    value,
		To:       &self.contract_addr,
		From:     from,
		Input:    input,
		Gas:      1000000,
		GasPrice: big.NewInt(0),
		Nonce:    senderNonce,
	})

	self.St.EndBlock()
	self.St.Commit()

	return res
}

func (self *ContractTest) AdvanceBlock(author *common.Address, rewardsStats *rewards_stats.RewardsStats) (ret *uint256.Int) {
	self.blk_n++
	if author == nil {
		self.St.BeginBlock(&vm.BlockInfo{})
	} else {
		self.St.BeginBlock(&vm.BlockInfo{*author, 0, 0, nil})
	}
	ret = self.St.DistributeRewards(rewardsStats)
	self.St.EndBlock()
	self.St.Commit()
	return
}

func (self *ContractTest) GetBalance(account *common.Address) *big.Int {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(account, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	return bal_actual
}

func (self *ContractTest) CheckContractBalance(balance *big.Int) {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&self.contract_addr, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	self.tc.Assert.Equal(balance, bal_actual)
}

func (self *ContractTest) GetNonce(account common.Address) *big.Int {
	nonce := big.NewInt(0)
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		nonce = account.Nonce
	})
	return nonce
}

func (self *ContractTest) GetDPOSReader() dpos.Reader {
	return self.SUT.DPOSDelayedReader(self.blk_n)
}

func (self *ContractTest) ExecuteAndCheck(from common.Address, value *big.Int, input []byte, exe_err, cons_err util.ErrorString) vm.ExecutionResult {
	res := self.execute(from, value, input)
	self.tc.Assert.Equal(cons_err, res.ConsensusErr)
	self.tc.Assert.Equal(exe_err, res.ExecutionErr)

	return res
}

func (self *ContractTest) BlockNumber() uint64 {
	return self.blk_n
}

func (self *ContractTest) End() {
	self.Statedb.Close()
	self.tc.Close()
}

func (self *ContractTest) GetJailedValidators() (jailed_validators []common.Address) {
	result := self.ExecuteAndCheck(tests.Addr(1), big.NewInt(0), self.Pack("getJailedValidators"), util.ErrorString(""), util.ErrorString(""))
	self.Unpack(&jailed_validators, "getJailedValidators", result.CodeRetval)
	return
}

func (self *ContractTest) Pack(name string, args ...interface{}) []byte {
	packed, err := self.abi.Pack(name, args...)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return packed
}

func (self *ContractTest) MethodId(name string) []byte {
	return self.abi.Methods[name].Id()
}

func (self *ContractTest) Unpack(v interface{}, name string, output []byte) error {
	err := self.abi.Unpack(v, name, output)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return err
}
