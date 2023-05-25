package test_integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/solidity"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
)

type GetUndelegationsRet struct {
	Undelegations []sol.DposInterfaceUndelegationData
	End           bool
}

type GetValidatorsRet struct {
	Validators []sol.DposInterfaceValidatorData
	End        bool
}

type GetDelegationsRet struct {
	Delegations []sol.DposInterfaceDelegationData
	End         bool
}

type GetValidatorRet struct {
	ValidatorInfo sol.DposInterfaceValidatorBasicInfo
}

type ClaimAllRewardsRet struct {
	End bool
}

type GenesisBalances = map[common.Address]*big.Int

var addr, addr_p = tests.Addr, tests.AddrP

type DposTest struct {
	Chain_cfg chain_config.ChainConfig
	st        state.StateTransition
	statedb   *state_db_rocksdb.DB
	tc        *tests.TestCtx
	SUT       *state.API
	blk_n     types.BlockNum
	dpos_addr common.Address
	abi       abi.ABI
}

func init_test(t *testing.T, cfg chain_config.ChainConfig) (tc tests.TestCtx, test DposTest) {
	tc = tests.NewTestCtx(t)
	test.init(&tc, cfg)
	return
}

func (self *DposTest) init(t *tests.TestCtx, cfg chain_config.ChainConfig) {
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

	self.st = self.SUT.GetStateTransition()
	self.dpos_addr = dpos.ContractAddress()
	self.abi, _ = abi.JSON(strings.NewReader(sol.TaraxaDposClientMetaData))
}

func (self *DposTest) execute(from common.Address, value *big.Int, input []byte) vm.ExecutionResult {
	senderNonce := self.GetNonce(from)
	senderNonce.Add(senderNonce, big.NewInt(1))

	self.blk_n++
	self.st.BeginBlock(&vm.BlockInfo{})

	res := self.st.ExecuteTransaction(&vm.Transaction{
		Value:    value,
		To:       &self.dpos_addr,
		From:     from,
		Input:    input,
		Gas:      1000000,
		GasPrice: big.NewInt(0),
		Nonce:    senderNonce,
	})

	self.st.EndBlock()
	self.st.Commit()
	return res
}

func (self *DposTest) AdvanceBlock(author *common.Address, rewardsStats *rewards_stats.RewardsStats, feesRewards *dpos.FeesRewards) (ret *uint256.Int) {
	self.blk_n++
	if author == nil {
		self.st.BeginBlock(&vm.BlockInfo{})
	} else {
		self.st.BeginBlock(&vm.BlockInfo{*author, 0, 0, nil})
	}
	ret = self.st.DistributeRewards(rewardsStats, feesRewards)
	self.st.EndBlock()
	self.st.Commit()
	return
}

func (self *DposTest) GetBalance(account common.Address) *big.Int {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	return bal_actual
}

func (self *DposTest) CheckContractBalance(balance *big.Int) {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&self.dpos_addr, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	self.tc.Assert.Equal(balance, bal_actual)
}

func (self *DposTest) GetNonce(account common.Address) *big.Int {
	nonce := big.NewInt(0)
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		nonce = account.Nonce
	})
	return nonce
}

func (self *DposTest) GetDPOSReader() dpos.Reader {
	return self.SUT.DPOSReader(self.blk_n)
}

func (self *DposTest) ExecuteAndCheck(from common.Address, value *big.Int, input []byte, exe_err, cons_err util.ErrorString) vm.ExecutionResult {
	res := self.execute(from, value, input)
	self.tc.Assert.Equal(cons_err, res.ConsensusErr)
	self.tc.Assert.Equal(exe_err, res.ExecutionErr)

	return res
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

func (self *DposTest) unpack(v interface{}, name string, output []byte) error {
	err := self.abi.Unpack(v, name, output)
	if err != nil {
		self.tc.Error(err)
		self.tc.FailNow()
	}
	return err
}

func generateKeyPair() (pubkey []byte, privkey *ecdsa.PrivateKey) {
	privkey, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(btcec.S256(), privkey.X, privkey.Y)
	return
}

func generateAddrAndProof() (addr common.Address, proof []byte) {
	pubkey, seckey := generateKeyPair()
	addr = common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])
	proof, _ = sign(keccak256.Hash(addr.Bytes()).Bytes(), seckey)
	return
}

// This is modified version of sign to match python implementation, do not use this outside of this package
func sign(hash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	if prv.Curve != btcec.S256() {
		return nil, fmt.Errorf("private key curve is not secp256k1")
	}
	sig, err := btcec.SignCompact(btcec.S256(), (*btcec.PrivateKey)(prv), hash, false)
	if err != nil {
		return nil, err
	}
	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := sig[0]
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
}

func initValidatorTrxsStats(validator common.Address, feesRewards *dpos.FeesRewards, trxFee *big.Int, trxsCount uint32) {
	f, _ := uint256.FromBig(trxFee)
	for i := uint32(0); i < trxsCount; i++ {
		feesRewards.AddTrxFeeReward(validator, f.Clone())
	}
}
