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

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/solidity"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
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

var (
	Big0                               = big.NewInt(0)
	Big1                               = big.NewInt(1)
	Big5                               = big.NewInt(5)
	Big10                              = big.NewInt(10)
	Big50                              = big.NewInt(50)
	TaraPrecision                      = big.NewInt(1e+18)
	DefaultBalance                     = bigutil.Mul(big.NewInt(5000000), TaraPrecision)
	DefaultEligibilityBalanceThreshold = bigutil.Mul(big.NewInt(1000000), TaraPrecision)
	DefaultVoteEligibilityBalanceStep  = bigutil.Mul(big.NewInt(1000), TaraPrecision)
	DefaultValidatorMaximumStake       = bigutil.Mul(big.NewInt(10000000), TaraPrecision)
	DefaultMinimumDeposit              = bigutil.Mul(big.NewInt(1000), TaraPrecision)

	DefaultChainCfg = chain_config.ChainConfig{
		ExecutionOptions: vm.ExecutionOpts{
			DisableNonceCheck:   true,
			EnableNonceSkipping: true,
		},
		BlockRewardsOptions: chain_config.BlockRewardsOpts{
			DisableBlockRewards:         false,
			DisableContractDistribution: false,
		},
		ETHChainConfig: params.ChainConfig{
			DAOForkBlock: types.BlockNumberNIL,
		},
		GenesisBalances: GenesisBalances{addr(1): DefaultBalance, addr(2): DefaultBalance, addr(3): DefaultBalance, addr(4): DefaultBalance, addr(5): DefaultBalance},
		DPOS: &dpos.Config{
			EligibilityBalanceThreshold: DefaultEligibilityBalanceThreshold,
			VoteEligibilityBalanceStep:  DefaultVoteEligibilityBalanceStep,
			ValidatorMaximumStake:       DefaultValidatorMaximumStake,
			MinimumDeposit:              DefaultMinimumDeposit,
			MaxBlockAuthorReward:        10,
			CommissionChangeDelta:       0,
			CommissionChangeFrequency:   0,
			DelegationDelay:             2,
			DelegationLockingPeriod:     4,
			BlocksPerYear:               365 * 24 * 60 * 15, // block every 4 seconds
			YieldPercentage:             20,
		},
	}
)

func init_test(t *testing.T, cfg chain_config.ChainConfig) (tc tests.TestCtx, test DposTest) {
	tc = tests.NewTestCtx(t)
	test.init(&tc, cfg)
	return
}

// When running test suite, it is somehow overriding default config so it must be copied...
// TODO: fix this
func CopyDefaulChainConfig() chain_config.ChainConfig {
	var new_cfg chain_config.ChainConfig

	new_cfg.ExecutionOptions.DisableNonceCheck = DefaultChainCfg.ExecutionOptions.DisableNonceCheck
	new_cfg.ExecutionOptions.EnableNonceSkipping = DefaultChainCfg.ExecutionOptions.EnableNonceSkipping

	new_cfg.BlockRewardsOptions.DisableBlockRewards = DefaultChainCfg.BlockRewardsOptions.DisableBlockRewards
	new_cfg.BlockRewardsOptions.DisableContractDistribution = DefaultChainCfg.BlockRewardsOptions.DisableContractDistribution

	new_cfg.ETHChainConfig.DAOForkBlock = DefaultChainCfg.ETHChainConfig.DAOForkBlock

	new_cfg.GenesisBalances = make(GenesisBalances)
	for k, v := range DefaultChainCfg.GenesisBalances {
		new_cfg.GenesisBalances[k] = v
	}

	new_cfg.DPOS = new(dpos.Config)
	new_cfg.DPOS.MaxBlockAuthorReward = DefaultChainCfg.DPOS.MaxBlockAuthorReward
	new_cfg.DPOS.CommissionChangeDelta = DefaultChainCfg.DPOS.CommissionChangeDelta
	new_cfg.DPOS.CommissionChangeFrequency = DefaultChainCfg.DPOS.CommissionChangeFrequency
	new_cfg.DPOS.ValidatorMaximumStake = DefaultChainCfg.DPOS.ValidatorMaximumStake
	new_cfg.DPOS.MinimumDeposit = DefaultChainCfg.DPOS.MinimumDeposit
	new_cfg.DPOS.DelegationLockingPeriod = DefaultChainCfg.DPOS.DelegationLockingPeriod
	new_cfg.DPOS.DelegationDelay = DefaultChainCfg.DPOS.DelegationDelay
	new_cfg.DPOS.EligibilityBalanceThreshold = DefaultChainCfg.DPOS.EligibilityBalanceThreshold
	new_cfg.DPOS.VoteEligibilityBalanceStep = DefaultChainCfg.DPOS.EligibilityBalanceThreshold
	new_cfg.DPOS.YieldPercentage = DefaultChainCfg.DPOS.YieldPercentage
	new_cfg.DPOS.BlocksPerYear = DefaultChainCfg.DPOS.BlocksPerYear
	new_cfg.DPOS.InitialValidators = DefaultChainCfg.DPOS.InitialValidators

	return new_cfg
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
	self.blk_n++
	self.st.BeginBlock(&vm.BlockInfo{})

	res := self.st.ExecuteTransaction(&vm.Transaction{
		Value:    value,
		To:       &self.dpos_addr,
		From:     from,
		Input:    input,
		Gas:      1000000,
		GasPrice: bigutil.Big0,
	})

	self.st.EndBlock(nil, nil, nil)
	self.st.Commit()
	return res
}

func (self *DposTest) AdvanceBlock(author *common.Address, rewardsStats *rewards_stats.RewardsStats, feesRewards *dpos.FeesRewards) {
	self.blk_n++
	if author == nil {
		self.st.BeginBlock(&vm.BlockInfo{})
	} else {
		self.st.BeginBlock(&vm.BlockInfo{*author, 0, 0, nil})
	}
	self.st.EndBlock(nil, rewardsStats, feesRewards)
	self.st.Commit()
}

func (self *DposTest) GetBalance(account common.Address) *big.Int {
	var bal_actual *big.Int
	self.SUT.ReadBlock(self.blk_n).GetAccount(&account, func(account state_db.Account) {
		bal_actual = account.Balance
	})
	return bal_actual
}

func (self *DposTest) GetDPOSReader() dpos.Reader {
	return self.SUT.DPOSReader(self.blk_n)
}

func (self *DposTest) ExecuteAndCheck(from common.Address, value *big.Int, input []byte, exe_err util.ErrorString, cons_err util.ErrorString) vm.ExecutionResult {
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

func initValidatorTxsStats(validator common.Address, feesRewards *dpos.FeesRewards, txFee *big.Int, txsCount uint32) {
	for i := uint32(0); i < txsCount; i++ {
		feesRewards.AddTxFeeReward(validator, txFee)
	}
}
