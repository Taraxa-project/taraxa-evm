package test_utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math/big"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/solidity"
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
	statedb       *state_db_rocksdb.DB
	tc            *tests.TestCtx
	SUT           *state.API
	blk_n         types.BlockNum
	slashing_addr common.Address
	abi           abi.ABI
}

func Init_test(contract_addr common.Address, t *testing.T, cfg chain_config.ChainConfig) (tc tests.TestCtx, test ContractTest) {
	tc = tests.NewTestCtx(t)
	test.init(contract_addr, &tc, cfg)
	return
}

func (self *ContractTest) init(contract_addr common.Address, t *tests.TestCtx, cfg chain_config.ChainConfig) {
	self.tc = t
	self.Chain_cfg = cfg

	self.statedb = new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: self.tc.DataDir(),
	})
	self.SUT = new(state.API).Init(
		self.statedb,
		func(num types.BlockNum) *big.Int { panic("unexpected") },
		&self.Chain_cfg,
		state.APIOpts{},
	)

	self.St = self.SUT.GetStateTransition()
	self.contract_addr = contract_addr
	self.abi, _ = abi.JSON(strings.NewReader(sol.TaraxaDposClientMetaData))
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

func (self *ContractTest) AdvanceBlock(author *common.Address, rewardsStats *rewards_stats.RewardsStats, feesRewards *dpos.FeesRewards) (ret *uint256.Int) {
	self.blk_n++
	if author == nil {
		self.St.BeginBlock(&vm.BlockInfo{})
	} else {
		self.St.BeginBlock(&vm.BlockInfo{*author, 0, 0, nil})
	}
	ret = self.St.DistributeRewards(rewardsStats, feesRewards)
	self.St.EndBlock()
	self.St.Commit()
	return
}

func (self *ContractTest) GetBalance(account common.Address) *big.Int {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
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
	return self.SUT.DPOSReader(self.blk_n)
}

func (self *ContractTest) ExecuteAndCheck(from common.Address, value *big.Int, input []byte, exe_err, cons_err util.ErrorString) vm.ExecutionResult {
	res := self.execute(from, value, input)
	self.tc.Assert.Equal(cons_err, res.ConsensusErr)
	self.tc.Assert.Equal(exe_err, res.ExecutionErr)

	return res
}

func (self *ContractTest) End() {
	self.statedb.Close()
	self.tc.Close()
}

func (self *ContractTest) Pack(name string, args ...interface{}) []byte {
	packed, err := self.abi.Pack(name, args...)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return packed
}

func (self *ContractTest) Unpack(v interface{}, name string, output []byte) error {
	err := self.abi.Unpack(v, name, output)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return err
}

func GenerateKeyPair() (pubkey []byte, privkey *ecdsa.PrivateKey) {
	privkey, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(btcec.S256(), privkey.X, privkey.Y)
	return
}
