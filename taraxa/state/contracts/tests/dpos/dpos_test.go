package dpos_tests

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	dpos_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/solidity"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	test_utils "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/tests"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"
	"github.com/btcsuite/btcd/btcec"
	"github.com/holiman/uint256"
)

// This strings should correspond to event signatures in ../solidity/dpos_contract_interface.sol file
var DelegatedEventHash = *keccak256.Hash([]byte("Delegated(address,address,uint256)"))
var UndelegatedEventHash = *keccak256.Hash([]byte("Undelegated(address,address,uint256)"))
var UndelegatedV2EventHash = *keccak256.Hash([]byte("UndelegatedV2(address,address,uint64,uint256)"))
var UndelegateConfirmedEventHash = *keccak256.Hash([]byte("UndelegateConfirmed(address,address,uint256)"))
var UndelegateConfirmedV2EventHash = *keccak256.Hash([]byte("UndelegateConfirmedV2(address,address,uint64,uint256)"))
var UndelegateCanceledEventHash = *keccak256.Hash([]byte("UndelegateCanceled(address,address,uint256)"))
var UndelegateCanceledV2EventHash = *keccak256.Hash([]byte("UndelegateCanceledV2(address,address,uint64,uint256)"))
var RedelegatedEventHash = *keccak256.Hash([]byte("Redelegated(address,address,address,uint256)"))
var RewardsClaimedEventHash = *keccak256.Hash([]byte("RewardsClaimed(address,address,uint256)"))
var CommissionRewardsClaimedEventHash = *keccak256.Hash([]byte("CommissionRewardsClaimed(address,address,uint256)"))
var CommissionSetEventHash = *keccak256.Hash([]byte("CommissionSet(address,uint16)"))
var ValidatorRegisteredEventHash = *keccak256.Hash([]byte("ValidatorRegistered(address)"))
var ValidatorInfoSetEventHash = *keccak256.Hash([]byte("ValidatorInfoSet(address)"))

type GetUndelegationsRet struct {
	Undelegations []dpos_sol.DposInterfaceUndelegationData
	End           bool
}

type GetUndelegationsV2Ret struct {
	UndelegationsV2 []dpos_sol.DposInterfaceUndelegationV2Data
	End             bool
}

type GetUndelegationV2Ret struct {
	UndelegationV2 dpos_sol.DposInterfaceUndelegationV2Data
}

type GetValidatorsRet struct {
	Validators []dpos_sol.DposInterfaceValidatorData
	End        bool
}

type GetTotalDelegationRet struct {
	TotalDelegation *big.Int
}

type GetDelegationsRet struct {
	Delegations []dpos_sol.DposInterfaceDelegationData
	End         bool
}

type GetValidatorRet struct {
	ValidatorInfo dpos_sol.DposInterfaceValidatorBasicInfo
}

type ClaimAllRewardsRet struct {
	End bool
}

var addr, addr_p = tests.Addr, tests.AddrP

type DposTest struct {
	Chain_cfg chain_config.ChainConfig
}

type GenesisBalances = map[common.Address]*big.Int

var (
	TaraPrecision                      = big.NewInt(1e+18)
	DefaultBalance                     = bigutil.Mul(big.NewInt(2050000000), TaraPrecision)
	DefaultEligibilityBalanceThreshold = bigutil.Mul(big.NewInt(1000000), TaraPrecision)
	DefaultVoteEligibilityBalanceStep  = bigutil.Mul(big.NewInt(1000), TaraPrecision)
	DefaultValidatorMaximumStake       = bigutil.Mul(big.NewInt(10000000), TaraPrecision)
	DefaultMinimumDeposit              = bigutil.Mul(big.NewInt(1000), TaraPrecision)
	DefaultVrfKey                      = common.RightPadBytes([]byte("0x0"), 32)

	DefaultChainCfg = chain_config.ChainConfig{
		GenesisBalances: GenesisBalances{addr(1): DefaultBalance, addr(2): DefaultBalance, addr(3): DefaultBalance, addr(4): DefaultBalance, addr(5): DefaultBalance},
		DPOS: chain_config.DPOSConfig{
			EligibilityBalanceThreshold: DefaultEligibilityBalanceThreshold,
			VoteEligibilityBalanceStep:  DefaultVoteEligibilityBalanceStep,
			ValidatorMaximumStake:       DefaultValidatorMaximumStake,
			MinimumDeposit:              DefaultMinimumDeposit,
			MaxBlockAuthorReward:        10,
			DagProposersReward:          50,
			CommissionChangeDelta:       0,
			CommissionChangeFrequency:   0,
			DelegationDelay:             2,
			DelegationLockingPeriod:     4,
			BlocksPerYear:               365 * 24 * 60 * 15, // block every 4 seconds
			YieldPercentage:             20,
		},
		Hardforks: chain_config.HardforksConfig{
			FixRedelegateBlockNum: 0,
			MagnoliaHf: chain_config.MagnoliaHfConfig{
				BlockNum: 0,
				JailTime: 5,
			},
			AspenHf: chain_config.AspenHfConfig{
				BlockNumPartOne: 0,
				BlockNumPartTwo: 0,
				// Max token supply is 12 Billion TARA -> 12e+9(12 billion) * 1e+18(tara precision)
				MaxSupply:        new(big.Int).Mul(big.NewInt(12e+9), big.NewInt(1e+18)),
				GeneratedRewards: big.NewInt(0),
			},
			CornusHf: chain_config.CornusHfConfig{
				BlockNum:                1000,
				DelegationLockingPeriod: 4,
				DagGasLimit:             100000,
				PbftGasLimit:            1000000,
			},
			SoleiroliaHf: chain_config.SoleiroliaHfConfig{
				BlockNum:       0,
				TrxMinGasPrice: 1,
				TrxMaxGasLimit: 1,
			},
		},
	}
)

// When running test suite, it is somehow overriding default config so it must be copied...
// TODO: fix this
func CopyDefaultChainConfig() chain_config.ChainConfig {
	var new_cfg chain_config.ChainConfig

	new_cfg.GenesisBalances = make(GenesisBalances)
	for k, v := range DefaultChainCfg.GenesisBalances {
		new_cfg.GenesisBalances[k] = v
	}

	new_cfg.DPOS.MaxBlockAuthorReward = DefaultChainCfg.DPOS.MaxBlockAuthorReward
	new_cfg.DPOS.DagProposersReward = DefaultChainCfg.DPOS.DagProposersReward
	new_cfg.DPOS.CommissionChangeDelta = DefaultChainCfg.DPOS.CommissionChangeDelta
	new_cfg.DPOS.CommissionChangeFrequency = DefaultChainCfg.DPOS.CommissionChangeFrequency
	new_cfg.DPOS.ValidatorMaximumStake = DefaultChainCfg.DPOS.ValidatorMaximumStake
	new_cfg.DPOS.MinimumDeposit = DefaultChainCfg.DPOS.MinimumDeposit
	new_cfg.DPOS.DelegationLockingPeriod = DefaultChainCfg.DPOS.DelegationLockingPeriod
	new_cfg.DPOS.DelegationDelay = DefaultChainCfg.DPOS.DelegationDelay
	new_cfg.DPOS.EligibilityBalanceThreshold = DefaultChainCfg.DPOS.EligibilityBalanceThreshold
	new_cfg.DPOS.VoteEligibilityBalanceStep = DefaultChainCfg.DPOS.VoteEligibilityBalanceStep
	new_cfg.DPOS.YieldPercentage = DefaultChainCfg.DPOS.YieldPercentage
	new_cfg.DPOS.BlocksPerYear = DefaultChainCfg.DPOS.BlocksPerYear
	new_cfg.DPOS.InitialValidators = DefaultChainCfg.DPOS.InitialValidators
	new_cfg.Hardforks = DefaultChainCfg.Hardforks

	return new_cfg
}

func GenerateKeyPair() (pubkey []byte, privkey *ecdsa.PrivateKey) {
	privkey, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(btcec.S256(), privkey.X, privkey.Y)
	return
}

func generateAddrAndProof() (addr common.Address, proof []byte) {
	pubkey, seckey := GenerateKeyPair()
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

func NewRewardsStats(author *common.Address) rewards_stats.RewardsStats {
	rewardsStats := rewards_stats.RewardsStats{}
	rewardsStats.BlockAuthor = *author
	rewardsStats.ValidatorsStats = make(map[common.Address]rewards_stats.ValidatorStats)

	return rewardsStats
}

func TestProof(t *testing.T) {
	pubkey, seckey := GenerateKeyPair()
	addr := common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])
	proof, _ := sign(addr.Hash().Bytes(), seckey)
	pubkey2, err := crypto.Ecrecover(addr.Hash().Bytes(), append(proof[:64], proof[64]-27))
	if err != nil {
		t.Errorf(err.Error())
	}
	if !bytes.Equal(pubkey, pubkey2) {
		t.Errorf("pubkey mismatch: want: %x have: %x", pubkey, pubkey2)
	}
	if common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:]) != addr {
		t.Errorf("pubkey mismatch: want: %x have: %x", addr, addr)
	}
}

func TestRegisterValidator(t *testing.T) {
	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	validator1_owner := addr(1)
	validator1_addr, validator1_proof := generateAddrAndProof()

	validator2_owner := addr(2)
	validator2_addr, validator2_proof := generateAddrAndProof()

	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	// Try to register same validator twice
	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	// Try to register with not enough balance
	test.ExecuteAndCheck(validator2_owner, bigutil.Add(DefaultBalance, big.NewInt(1)), test.Pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.NewErrorString(vm.ErrInsufficientBalanceForTransfer))
	// Try to register with wrong proof
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrWrongProof, util.ErrorString(""))
}

func TestDelegate(t *testing.T) {
	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	// Try to delegate to not existent validator
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("delegate", addr(2)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit))
}

func TestDelegateMinMax(t *testing.T) {
	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	delegation := bigutil.Mul(big.NewInt(5000000), TaraPrecision)

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	test.ExecuteAndCheck(addr(1), bigutil.Sub(delegation, DefaultMinimumDeposit), test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance := bigutil.Sub(delegation, DefaultMinimumDeposit)
	totalBalance.Add(totalBalance, DefaultMinimumDeposit)
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(addr(2), bigutil.Sub(DefaultMinimumDeposit, big.NewInt(1)), test.Pack("delegate", val_addr), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(3), bigutil.Sub(delegation, DefaultMinimumDeposit), test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance.Add(totalBalance, bigutil.Sub(delegation, DefaultMinimumDeposit))
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(addr(2), bigutil.Sub(delegation, DefaultMinimumDeposit), test.Pack("delegate", val_addr), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), DefaultMinimumDeposit, test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance.Add(totalBalance, DefaultMinimumDeposit)
	test.CheckContractBalance(totalBalance)
}

func TestRedelegate(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	validator1_owner := addr(1)
	validator1_addr, validator1_proof := generateAddrAndProof()

	validator2_owner := addr(2)
	validator2_addr, validator2_proof := generateAddrAndProof()

	reg_res := test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(reg_res.Logs), 2)
	tc.Assert.Equal(reg_res.Logs[0].Topics[0], ValidatorRegisteredEventHash)
	tc.Assert.Equal(reg_res.Logs[1].Topics[0], DelegatedEventHash)
	test.CheckContractBalance(DefaultMinimumDeposit)

	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	totalBalance := bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit)
	test.CheckContractBalance(totalBalance)
	redelegate_res := test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(redelegate_res.Logs), 1)
	tc.Assert.Equal(redelegate_res.Logs[0].Topics[0], RedelegatedEventHash)
	test.CheckContractBalance(totalBalance)

	//Validator 1 does not exist as we withdraw all stake
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))

	vali1_new_delegation := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))
	test.ExecuteAndCheck(validator1_owner, vali1_new_delegation, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	totalBalance.Add(totalBalance, vali1_new_delegation)
	test.CheckContractBalance(totalBalance)
	// Validator to does not exist
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, addr(3), DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Validator from does not exist
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", addr(3), validator1_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Non existent delegation
	test.ExecuteAndCheck(addr(3), big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	// InsufficientDelegation
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, bigutil.Add(vali1_new_delegation, big.NewInt(1))), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// Validator 1 does not exist as we withdraw all stake
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Validator can not redelegate to same validator
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator1_addr, DefaultMinimumDeposit), dpos.ErrSameValidator, util.ErrorString(""))

	// Check for negative and zero redelegation
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, big.NewInt(0)), dpos.ErrInvalidRedelegation, util.ErrorString(""))

	test.CheckContractBalance(totalBalance)
}

func TestRedelegateMinMax(t *testing.T) {
	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)

	validator2_addr, validator2_proof := generateAddrAndProof()
	validator2_owner := addr(2)

	init_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))

	delegation := bigutil.Mul(big.NewInt(5000000), TaraPrecision)

	test.ExecuteAndCheck(validator1_owner, init_stake, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(validator2_owner, init_stake, test.Pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	totalBalance := bigutil.Add(init_stake, init_stake)
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, bigutil.Add(DefaultMinimumDeposit, big.NewInt(1))), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(validator2_owner, bigutil.Sub(delegation, init_stake), test.Pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance.Add(totalBalance, bigutil.Sub(delegation, init_stake))
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(addr(3), delegation, test.Pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance.Add(totalBalance, delegation)
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, big.NewInt(1)), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
}
func TestUndelegate(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	undelegate_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(undelegate_res.Logs), 1)
	tc.Assert.Equal(undelegate_res.Logs[0].Topics[0], UndelegatedEventHash)
	totalBalance := DefaultMinimumDeposit
	test.CheckContractBalance(totalBalance)

	// Validator exists - should not be deleted yet
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrExistentUndelegation as one undelegation request already exists
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrExistentUndelegation, util.ErrorString(""))

	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("cancelUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrExistentValidator
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	// NonExistentValidator
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", delegator_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// NonExistentDelegation
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))

	// ErrInsufficientDelegation
	test.ExecuteAndCheck(delegator_addr, DefaultMinimumDeposit, test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance.Add(totalBalance, DefaultMinimumDeposit)
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("undelegate", val_addr, bigutil.Add(DefaultMinimumDeposit, big.NewInt(1))), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
}

func TestUndelegateV2(t *testing.T) {
	cfg := CopyDefaultChainConfig()
	cfg.Hardforks.CornusHf.BlockNum = 0
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultValidatorMaximumStake, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	totalBalance := DefaultValidatorMaximumStake
	test.CheckContractBalance(totalBalance)

	// Create 4 undelegations from the same validator
	var undelegations_blocks []uint64
	for idx := uint64(1); idx <= 4; idx++ {
		undelegate_v2_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
		undelegations_blocks = append(undelegations_blocks, test.BlockNumber()+uint64(cfg.DPOS.DelegationLockingPeriod))

		tc.Assert.Equal(len(undelegate_v2_res.Logs), 1)
		tc.Assert.Equal(undelegate_v2_res.Logs[0].Topics[0], UndelegatedV2EventHash)
		test.CheckContractBalance(totalBalance)

		undelegation_id_parsed := new(uint64)
		test.Unpack(undelegation_id_parsed, "undelegateV2", undelegate_v2_res.CodeRetval)
		tc.Assert.Equal(idx, *undelegation_id_parsed)
	}

	// Cancel undelegation with id == 2
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("cancelUndelegateV2", val_addr, uint64(2)), util.ErrorString(""), util.ErrorString(""))

	// Confirm undelegation with id == 3
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("cancelUndelegateV2", val_addr, uint64(3)), util.ErrorString(""), util.ErrorString(""))

	// Get undelegations one by one
	for idx := uint64(1); idx <= 4; idx++ {
		if idx == 2 || idx == 3 {
			test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getUndelegationV2", val_owner, val_addr, uint64(10)), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
			continue
		}

		get_undelegation_v2_result := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getUndelegationV2", val_owner, val_addr, idx), util.ErrorString(""), util.ErrorString(""))
		get_undelegation_v2_parsed_result := new(GetUndelegationV2Ret)
		test.Unpack(get_undelegation_v2_parsed_result, "getUndelegationV2", get_undelegation_v2_result.CodeRetval)

		tc.Assert.Equal(idx, get_undelegation_v2_parsed_result.UndelegationV2.UndelegationId)
		tc.Assert.Equal(DefaultMinimumDeposit, get_undelegation_v2_parsed_result.UndelegationV2.UndelegationData.Stake)
		tc.Assert.Equal(val_addr, get_undelegation_v2_parsed_result.UndelegationV2.UndelegationData.Validator)
		tc.Assert.Equal(undelegations_blocks[idx-1], get_undelegation_v2_parsed_result.UndelegationV2.UndelegationData.Block)
	}
}

// In pre magnolia hardfork code, validator was deleted if his total_stake & rewards_pool == 0
// In post magnolia hardfork code, validator was deleted if his total_stake & rewards_pool & ongoing undelegations_count == 0
func TestPreMagnoliaHfUndelegate(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.MagnoliaHf.BlockNum = 1000

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()
	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	undelegate_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(undelegate_res.Logs), 1)
	tc.Assert.Equal(undelegate_res.Logs[0].Topics[0], UndelegatedEventHash)
	test.CheckContractBalance(DefaultMinimumDeposit)

	// Validator does not exist - was already deleted
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// ErrNonExistentValidator
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("cancelUndelegate", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// NonExistentValidator as it was deleted
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestMagnoliaHardfork(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.MagnoliaHf.BlockNum = 25

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	validator1_owner := addr(1)
	validator1_addr, validator1_proof := generateAddrAndProof()

	validator2_owner := addr(2)
	validator2_addr, validator2_proof := generateAddrAndProof()

	// Test pre-magnolia hardfork behaviour
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)

	test.ExecuteAndCheck(validator2_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_balance := bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit)
	test.CheckContractBalance(total_balance)

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("undelegate", validator1_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(total_balance)

	// ErrNonExistentValidator - validator was already deleted after undelegate
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// ErrNonExistentValidator - validator was already deleted after undelegate
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("cancelUndelegate", validator1_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// Advance 2 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("confirmUndelegate", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	total_balance = bigutil.Sub(total_balance, DefaultMinimumDeposit)
	test.CheckContractBalance(total_balance)

	// Register the same validator
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_balance = bigutil.Add(total_balance, DefaultMinimumDeposit)
	test.CheckContractBalance(total_balance)

	validator_raw := test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(uint16(0), validator.ValidatorInfo.UndelegationsCount)

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(total_balance)

	// ErrNonExistentValidator - validator was already deleted after redelegate
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// Register the same validator
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_balance = bigutil.Add(total_balance, DefaultMinimumDeposit)

	// Test post-magnolia hardfork behaviour

	// Advance few block so we are sure the current block already passed hardfork block num
	for i := uint64(0); i < cfg.Hardforks.MagnoliaHf.BlockNum; i++ {
		test.AdvanceBlock(nil, nil)
	}

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("undelegate", validator1_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(total_balance)

	// Validator still exists
	validator_raw = test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(uint16(1), validator.ValidatorInfo.UndelegationsCount)

	// Ok
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("cancelUndelegate", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(total_balance)

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("undelegate", validator1_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))

	// Advance 4 more rounds - delegation locking periods == 4
	for i := 0; i < 4; i++ {
		test.AdvanceBlock(nil, nil)
	}

	// Validator still exists
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("confirmUndelegate", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	total_balance = bigutil.Sub(total_balance, DefaultMinimumDeposit)
	test.CheckContractBalance(total_balance)

	// ErrNonExistentValidator - validator was deleted after confirmUndelegate
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// Register the same validator
	test.ExecuteAndCheck(validator1_owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_balance = bigutil.Add(total_balance, DefaultMinimumDeposit)
	test.CheckContractBalance(total_balance)
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("reDelegate", validator1_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(total_balance)

	// ErrNonExistentValidator - validator was already deleted after redelegate
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("getValidator", validator1_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestCornusHardfork(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.CornusHf.BlockNum = 10

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	val_owner2 := addr(2)
	val_addr2, proof2 := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultEligibilityBalanceThreshold, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultEligibilityBalanceThreshold)
	test.ExecuteAndCheck(val_owner2, DefaultEligibilityBalanceThreshold, test.Pack("registerValidator", val_addr2, proof2, DefaultVrfKey, uint16(10), "test3", "test3"), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("delegate", val_addr2), util.ErrorString(""), util.ErrorString(""))

	// ErrMethodNotSupported
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr, uint64(1)), dpos.ErrMethodNotSupported, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("cancelUndelegateV2", val_addr, uint64(1)), dpos.ErrMethodNotSupported, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), dpos.ErrMethodNotSupported, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getUndelegationsV2", val_addr, uint32(0)), dpos.ErrMethodNotSupported, util.ErrorString(""))

	undelegate_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(undelegate_res.Logs), 1)
	tc.Assert.Equal(undelegate_res.Logs[0].Topics[0], UndelegatedEventHash)

	// ErrExistentUndelegation
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrExistentUndelegation, util.ErrorString(""))

	// Pass cornus hf block num
	for test.BlockNumber() < cfg.Hardforks.CornusHf.BlockNum {
		test.AdvanceBlock(nil, nil)
	}

	undelegate_v2_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(undelegate_v2_res.Logs), 1)
	tc.Assert.Equal(undelegate_v2_res.Logs[0].Topics[0], UndelegatedV2EventHash)
	undelegation_id_parsed := new(uint64)
	test.Unpack(undelegation_id_parsed, "undelegateV2", undelegate_v2_res.CodeRetval)
	tc.Assert.Equal(uint64(1), *undelegation_id_parsed)

	undelegate_v2_res2 := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegateV2", val_addr2, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(undelegate_v2_res2.Logs), 1)
	tc.Assert.Equal(undelegate_v2_res2.Logs[0].Topics[0], UndelegatedV2EventHash)
	undelegation_id_parsed2 := new(uint64)
	test.Unpack(undelegation_id_parsed2, "undelegateV2", undelegate_v2_res2.CodeRetval)
	tc.Assert.Equal(uint64(2), *undelegation_id_parsed2)

	// Confirm V1 undelegation
	confirm_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(confirm_res.Logs), 1)
	tc.Assert.Equal(confirm_res.Logs[0].Topics[0], UndelegateConfirmedEventHash)

	// Get undelegation's id through getUndelegationsV2
	get_undelegations_v2_result := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getUndelegationsV2", val_owner, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	get_undelegations_v2_result_parsed := new(GetUndelegationsV2Ret)
	test.Unpack(get_undelegations_v2_result_parsed, "getUndelegationsV2", get_undelegations_v2_result.CodeRetval)
	tc.Assert.Equal(2, len(get_undelegations_v2_result_parsed.UndelegationsV2))
	tc.Assert.Equal(true, get_undelegations_v2_result_parsed.End)
	undelegation_id := get_undelegations_v2_result_parsed.UndelegationsV2[0].UndelegationId
	tc.Assert.Equal(*undelegation_id_parsed, undelegation_id)

	// Advance 2 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	// Confirm V2 undelegation
	confirm_res = test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr, undelegation_id), util.ErrorString(""), util.ErrorString(""))
	confirm_res2 := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr2, undelegation_id_parsed2), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(len(confirm_res.Logs), 1)
	tc.Assert.Equal(confirm_res.Logs[0].Topics[0], UndelegateConfirmedV2EventHash)
	tc.Assert.Equal(len(confirm_res2.Logs), 1)
	tc.Assert.Equal(confirm_res2.Logs[0].Topics[0], UndelegateConfirmedV2EventHash)
}

func TestCornusHardforkLockingPeriod(t *testing.T) {
	cfg := CopyDefaultChainConfig()
	cfg.Hardforks.CornusHf.BlockNum = 5
	cfg.Hardforks.CornusHf.DelegationLockingPeriod = 100
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultValidatorMaximumStake, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	totalBalance := DefaultValidatorMaximumStake
	test.CheckContractBalance(totalBalance)

	// Create undelegation before cornus hardfork
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	undelegate1_expected_lockup_block := test.BlockNumber() + uint64(cfg.DPOS.DelegationLockingPeriod)

	get_undelegations_result := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getUndelegations", val_owner, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	get_undelegations_parsed_result := new(GetUndelegationsRet)
	test.Unpack(get_undelegations_parsed_result, "getUndelegations", get_undelegations_result.CodeRetval)
	tc.Assert.Equal(1, len(get_undelegations_parsed_result.Undelegations))
	tc.Assert.Equal(true, get_undelegations_parsed_result.End)
	tc.Assert.Equal(undelegate1_expected_lockup_block, get_undelegations_parsed_result.Undelegations[0].Block)

	// Pass cornus hardfork
	tc.Assert.Less(test.BlockNumber(), cfg.Hardforks.CornusHf.BlockNum)
	for i := test.BlockNumber(); i < cfg.Hardforks.CornusHf.BlockNum; i++ {
		test.AdvanceBlock(nil, nil)
	}

	// Create undelegation after cornus hardfork
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	undelegate2_expected_id := uint64(1)
	undelegate2_expected_lockup_block := test.BlockNumber() + uint64(cfg.Hardforks.CornusHf.DelegationLockingPeriod)

	get_undelegation2_result := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getUndelegationV2", val_owner, val_addr, undelegate2_expected_id), util.ErrorString(""), util.ErrorString(""))
	get_undelegation2_parsed_result := new(GetUndelegationV2Ret)
	test.Unpack(get_undelegation2_parsed_result, "getUndelegationV2", get_undelegation2_result.CodeRetval)
	tc.Assert.Equal(undelegate2_expected_id, get_undelegation2_parsed_result.UndelegationV2.UndelegationId)
	tc.Assert.Equal(undelegate2_expected_lockup_block, get_undelegation2_parsed_result.UndelegationV2.UndelegationData.Block)
}

func TestConfirmUndelegate(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(addr(2), big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	totalBalance := DefaultMinimumDeposit
	test.CheckContractBalance(totalBalance)

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegate", val_addr), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	// Validator should not be deleted yet
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrLockedUndelegation
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegate", val_addr), dpos.ErrLockedUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	// Advance 2 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	confirm_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// TODO: values are equal(0) but big.nat differs in underlying big.Int objects ???
	// totalBalance = bigutil.Sub(totalBalance, DefaultMinimumDeposit)
	//test.CheckContractBalance(totalBalance)
	tc.Assert.Equal(len(confirm_res.Logs), 1)
	tc.Assert.Equal(confirm_res.Logs[0].Topics[0], UndelegateConfirmedEventHash)

	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestConfirmUndelegateV2(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.CornusHf.BlockNum = 0

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(addr(2), big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	totalBalance := DefaultMinimumDeposit
	test.CheckContractBalance(totalBalance)

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr, uint64(1)), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	undelegate_v2_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
	undelegation_id_parsed := new(uint64)
	test.Unpack(undelegation_id_parsed, "undelegateV2", undelegate_v2_res.CodeRetval)
	tc.Assert.Equal(uint64(1), *undelegation_id_parsed)

	// Validator should not be deleted yet
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrLockedUndelegation
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr, *undelegation_id_parsed), dpos.ErrLockedUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	// Advance 2 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr, *undelegation_id_parsed+uint64(1)), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)

	confirm_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegateV2", val_addr, *undelegation_id_parsed), util.ErrorString(""), util.ErrorString(""))

	// TODO: values are equal(0) but big.nat differs in underlying big.Int objects ???
	// totalBalance = bigutil.Sub(totalBalance, DefaultMinimumDeposit)
	//test.CheckContractBalance(totalBalance)
	tc.Assert.Equal(len(confirm_res.Logs), 1)
	tc.Assert.Equal(confirm_res.Logs[0].Topics[0], UndelegateConfirmedV2EventHash)

	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestCancelUndelegate(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("cancelUndelegate", val_addr), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	// Undelegate and check if validator's total stake was increased
	test.ExecuteAndCheck(delegator_addr, DefaultMinimumDeposit, test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance := bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit)
	test.CheckContractBalance(totalBalance)
	validator_raw := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit), validator.ValidatorInfo.TotalStake)

	// Undelegate and check if validator's total stake was decreased
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(DefaultMinimumDeposit, validator.ValidatorInfo.TotalStake)

	// Cancel undelegate and check if validator's total stake was increased again
	cancel_res := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("cancelUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
	tc.Assert.Equal(len(cancel_res.Logs), 1)
	tc.Assert.Equal(cancel_res.Logs[0].Topics[0], UndelegateCanceledEventHash)
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit), validator.ValidatorInfo.TotalStake)

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("cancelUndelegate", val_addr), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
}

func TestCancelUndelegateV2(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.Hardforks.CornusHf.BlockNum = 0

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("cancelUndelegateV2", val_addr, uint64(1)), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(DefaultMinimumDeposit)
	// Undelegate and check if validator's total stake was increased
	test.ExecuteAndCheck(delegator_addr, DefaultMinimumDeposit, test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	totalBalance := bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit)
	test.CheckContractBalance(totalBalance)
	validator_raw := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit), validator.ValidatorInfo.TotalStake)

	// Undelegate and check if validator's total stake was decreased
	undelegate_v2_res := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("undelegateV2", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
	undelegation_id_parsed := new(uint64)
	test.Unpack(undelegation_id_parsed, "undelegateV2", undelegate_v2_res.CodeRetval)
	tc.Assert.Equal(uint64(1), *undelegation_id_parsed)

	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(DefaultMinimumDeposit, validator.ValidatorInfo.TotalStake)

	// Cancel undelegate and check if validator's total stake was increased again
	cancel_res := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("cancelUndelegateV2", val_addr, *undelegation_id_parsed), util.ErrorString(""), util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
	tc.Assert.Equal(len(cancel_res.Logs), 1)
	tc.Assert.Equal(cancel_res.Logs[0].Topics[0], UndelegateCanceledV2EventHash)
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(bigutil.Add(DefaultMinimumDeposit, DefaultMinimumDeposit), validator.ValidatorInfo.TotalStake)

	// ErrNonExistentUndelegation
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("cancelUndelegateV2", val_addr, *undelegation_id_parsed), dpos.ErrNonExistentUndelegation, util.ErrorString(""))
	test.CheckContractBalance(totalBalance)
}

func TestUndelegateMin(t *testing.T) {
	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("undelegate", val_addr, big.NewInt(1)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), bigutil.Mul(DefaultMinimumDeposit, big.NewInt(3)), test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), big.NewInt(0), test.Pack("undelegate", val_addr, bigutil.Mul(DefaultMinimumDeposit, big.NewInt(2))), util.ErrorString(""), util.ErrorString(""))
}

func TestYieldCurveAspenHf(t *testing.T) {
	cfg := CopyDefaultChainConfig()
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	var yield_curve dpos.YieldCurve
	yield_curve.Init(cfg)

	// yield = (max supply - total supply) / total supply
	// block reward = yield * total stake / blocks per year

	// max supply has hardcoded value of 12 Billion TARA
	// total supply = 10 Billion, total stake = 1 Billion, expected yield == 20%
	total_supply := new(uint256.Int).Mul(uint256.NewInt(10e+9), uint256.NewInt(1e+18))
	total_stake := new(uint256.Int).Mul(uint256.NewInt(1e+9), uint256.NewInt(1e+18))
	expected_yield := uint256.NewInt(200000)
	expected_block_reward := calculateExpectedBlockReward(total_stake, expected_yield, cfg)
	block_reward, yield := yield_curve.CalculateBlockReward(total_stake, total_supply)

	tc.Assert.Equal(expected_block_reward, block_reward)
	tc.Assert.Equal(expected_yield, yield)

	// max supply = 12 Billion, total supply = 11 Billion, total stake = 1 Billion, expected yield == 9,0909%
	total_supply = new(uint256.Int).Mul(uint256.NewInt(11e+9), uint256.NewInt(1e+18))
	expected_yield = uint256.NewInt(90909)
	expected_block_reward = calculateExpectedBlockReward(total_stake, expected_yield, cfg)
	block_reward, yield = yield_curve.CalculateBlockReward(total_stake, total_supply)

	tc.Assert.Equal(expected_block_reward, block_reward)
	tc.Assert.Equal(expected_yield, yield)

	// max supply = 12 Billion, total supply = 11.5 Billion, total stake = 1 Billion, expected yield == 4,3478%
	total_supply = new(uint256.Int).Mul(uint256.NewInt(115e+8), uint256.NewInt(1e+18))
	expected_yield = uint256.NewInt(43478)
	expected_block_reward = calculateExpectedBlockReward(total_stake, expected_yield, cfg)
	block_reward, yield = yield_curve.CalculateBlockReward(total_stake, total_supply)

	tc.Assert.Equal(expected_block_reward, block_reward)
	tc.Assert.Equal(expected_yield, yield)

	// max supply = 12 Billion, total supply = 12 Billion, total stake = 1 Billion, expected yield == 0%
	total_supply = new(uint256.Int).Mul(uint256.NewInt(12e+9), uint256.NewInt(1e+18))
	expected_yield = uint256.NewInt(0)
	expected_block_reward = calculateExpectedBlockReward(total_stake, expected_yield, cfg)
	block_reward, yield = yield_curve.CalculateBlockReward(total_stake, total_supply)

	tc.Assert.Equal(expected_block_reward, block_reward)
	tc.Assert.Equal(expected_yield, yield)
}

func calculateExpectedBlockReward(total_stake *uint256.Int, expected_yield *uint256.Int, cfg chain_config.ChainConfig) *uint256.Int {
	expected_block_reward := new(uint256.Int).Mul(total_stake, expected_yield)
	expected_block_reward.Div(expected_block_reward, new(uint256.Int).Mul(dpos.YieldFractionDecimalPrecision, uint256.NewInt(uint64(cfg.DPOS.BlocksPerYear))))
	return expected_block_reward
}

func TestAspenHf(t *testing.T) {
	// Test if generated block reward changed from fixed yield to the new dynamic yield curve
	cfg := CopyDefaultChainConfig()
	cfg.Hardforks.AspenHf.BlockNumPartOne = 5
	cfg.Hardforks.AspenHf.BlockNumPartTwo = 10
	cfg.Hardforks.AspenHf.GeneratedRewards = bigutil.Mul(big.NewInt(5000000), big.NewInt(1e18)) // 5M TARA

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	total_supply := big.NewInt(0)
	for _, balance := range cfg.GenesisBalances {
		total_supply.Add(total_supply, balance)
	}
	total_supply.Add(total_supply, cfg.Hardforks.AspenHf.GeneratedRewards)

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)
	validator1_commission := uint16(500) // 5%
	delegator1_stake := DefaultValidatorMaximumStake

	// Creates single validator
	test.ExecuteAndCheck(validator1_owner, delegator1_stake, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, validator1_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake := delegator1_stake

	// Empty rewards statistics
	trxFee := bigutil.Div(TaraPrecision, big.NewInt(1000)) //  0.001 TARA
	tmp_rewards_stats := NewRewardsStats(&validator1_addr)

	validator1_stats := rewards_stats.ValidatorStats{}
	validator1_stats.DagBlocksCount = 10
	validator1_stats.VoteWeight = 1
	validator1_stats.FeesRewards = big.NewInt(int64(validator1_stats.DagBlocksCount))
	validator1_stats.FeesRewards.Mul(validator1_stats.FeesRewards, trxFee)
	tmp_rewards_stats.ValidatorsStats[validator1_addr] = validator1_stats

	tmp_rewards_stats.TotalDagBlocksCount = validator1_stats.DagBlocksCount
	tmp_rewards_stats.TotalVotesWeight = 1
	tmp_rewards_stats.MaxVotesWeight = 1

	txsNum := big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount))
	txsFees := bigutil.Mul(trxFee, txsNum)

	contract_balance := new(big.Int).Set(total_stake)
	// Advance couple of blocks - pre aspen.PartTwo hf with fixed yield
	for block_n := test.BlockNumber(); block_n < cfg.Hardforks.AspenHf.BlockNumPartTwo-1; block_n++ {
		expected_reward := bigutil.Mul(total_stake, big.NewInt(int64(test.Chain_cfg.DPOS.YieldPercentage)))
		expected_reward = bigutil.Div(expected_reward, bigutil.Mul(big.NewInt(100), big.NewInt(int64(test.Chain_cfg.DPOS.BlocksPerYear))))

		reward := test.AdvanceBlock(&validator1_addr, &tmp_rewards_stats).ToBig()
		tc.Assert.Equal(expected_reward, reward)

		contract_balance.Add(contract_balance, reward)
		contract_balance.Add(contract_balance, txsFees)
		test.CheckContractBalance(contract_balance)

		if block_n >= cfg.Hardforks.AspenHf.BlockNumPartOne-1 {
			total_supply.Add(total_supply, reward)
		}
	}

	// Expected block reward
	var yield_curve dpos.YieldCurve
	yield_curve.Init(cfg)

	// Advance couple of blocks - after aspen.PartTwo hf with dynamic yield
	for block_n := test.BlockNumber(); block_n < cfg.Hardforks.AspenHf.BlockNumPartTwo+20; block_n++ {
		total_supply_uin256, _ := uint256.FromBig(total_supply)
		total_stake_uin256, _ := uint256.FromBig(total_stake)
		expected_reward_uint256, _ := yield_curve.CalculateBlockReward(total_stake_uin256, total_supply_uin256)
		expected_reward := expected_reward_uint256.ToBig()

		reward := test.AdvanceBlock(&validator1_addr, &tmp_rewards_stats)
		reward_big := reward.ToBig()
		tc.Assert.Equal(expected_reward, reward_big)

		contract_balance.Add(contract_balance, reward_big)
		contract_balance.Add(contract_balance, txsFees)
		total_supply.Add(total_supply, reward_big)
		test.CheckContractBalance(contract_balance)
	}

	// Advance cfg.DPOS.DelegationDelay blocks and do not add rewards to the total_supply to make it equal to the test.GetDPOSReader().GetTotalSupply().
	// test.GetDPOSReader().GetTotalSupply() returns delayed data by cfg.DPOS.DelegationDelay blocks so after
	for idx := uint32(0); idx < cfg.DPOS.DelegationDelay; idx++ {
		test.AdvanceBlock(&validator1_addr, &tmp_rewards_stats)
	}

	tc.Assert.Equal(total_supply, test.GetDPOSReader().GetTotalSupply())
}

func TestRewardsAndCommission(t *testing.T) {
	cfg := CopyDefaultChainConfig()

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	trxFee := bigutil.Div(TaraPrecision, big.NewInt(1000)) //  0.001 TARA

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)
	validator1_commission := uint16(500) // 5%
	delegator1_addr := validator1_owner
	delegator1_stake := DefaultMinimumDeposit

	validator2_addr, validator2_proof := generateAddrAndProof()
	validator2_owner := addr(2)
	validator2_commission := uint16(200) // 2%
	delegator2_addr := validator2_owner
	delegator2_stake := DefaultMinimumDeposit

	delegator3_addr := addr(3)
	delegator3_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(4))

	validator4_addr, validator4_proof := generateAddrAndProof()
	validator4_owner := addr(4)
	validator4_commission := uint16(0) // 0%
	delegator4_addr := validator4_owner
	delegator4_stake := DefaultMinimumDeposit

	validator5_addr, validator5_proof := generateAddrAndProof()
	validator5_owner := addr(5)
	validator5_commission := uint16(10000) //100%
	// delegator5_addr := validator5_owner
	delegator5_stake := DefaultMinimumDeposit

	/*
		Simulate scenario when we have:

		  - total unique trxs count == 40
			- validator 1:
					- stake == 12.5% (from total stake)
					- he delegates to himself those 12.5%
					- added 8 unique trxs
					- 1 vote
			- validator 2:
					- stake == 62.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- added 32 unique trxs
					- 1 vote
			- delegator 3:
					- delegated 50% (from total stake) to validator 2
			- validator 4:
					- stake == 12.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- 1 vote
			- validator 5:
					- stake == 12.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- 1 vote

		After every participant claims his rewards:

			- block author (validator 1) - added 7 votes => bonus 1 vote
			- delegator 1(validator 1) gets 100 % from validator1_rewards
			- delegator 2(validator 2) gets 20 % from validator2_rewards
			- delegator 3 gets 80 % from validator2_rewards
			- delegator 4 gets 100% reward for 1 vote
	*/

	// Creates validators & delegators
	test.ExecuteAndCheck(validator1_owner, delegator1_stake, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, validator1_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake := delegator1_stake

	test.ExecuteAndCheck(validator2_owner, delegator2_stake, test.Pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, validator2_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator2_stake)

	test.ExecuteAndCheck(delegator3_addr, delegator3_stake, test.Pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator3_stake)

	test.ExecuteAndCheck(validator4_owner, delegator4_stake, test.Pack("registerValidator", validator4_addr, validator4_proof, DefaultVrfKey, validator4_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator4_stake)

	test.ExecuteAndCheck(validator5_owner, delegator5_stake, test.Pack("registerValidator", validator5_addr, validator5_proof, DefaultVrfKey, validator5_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator5_stake)

	test.CheckContractBalance(total_stake)

	// Simulated rewards statistics
	tmp_rewards_stats := NewRewardsStats(&validator1_addr)

	validator1_stats := rewards_stats.ValidatorStats{}
	validator1_stats.DagBlocksCount = 8
	validator1_stats.VoteWeight = 1
	validator1_stats.FeesRewards = big.NewInt(int64(validator1_stats.DagBlocksCount))
	validator1_stats.FeesRewards.Mul(validator1_stats.FeesRewards, trxFee)
	tmp_rewards_stats.ValidatorsStats[validator1_addr] = validator1_stats

	validator2_stats := rewards_stats.ValidatorStats{}
	validator2_stats.DagBlocksCount = 32
	validator2_stats.VoteWeight = 5
	validator2_stats.FeesRewards = big.NewInt(int64(validator2_stats.DagBlocksCount))
	validator2_stats.FeesRewards.Mul(validator2_stats.FeesRewards, trxFee)
	tmp_rewards_stats.ValidatorsStats[validator2_addr] = validator2_stats

	validator4_stats := rewards_stats.ValidatorStats{}
	validator4_stats.VoteWeight = 1
	tmp_rewards_stats.ValidatorsStats[validator4_addr] = validator4_stats

	tmp_rewards_stats.TotalDagBlocksCount = validator1_stats.DagBlocksCount + validator2_stats.DagBlocksCount
	tmp_rewards_stats.TotalVotesWeight = 7
	tmp_rewards_stats.MaxVotesWeight = 8

	// Advance block
	reward := test.AdvanceBlock(&validator1_addr, &tmp_rewards_stats).ToBig()
	totalBalance := bigutil.Add(total_stake, reward)
	numberOfTrxs := new(big.Int)
	numberOfTrxs.SetUint64(uint64(tmp_rewards_stats.TotalDagBlocksCount))
	totalBalance.Add(totalBalance, bigutil.Mul(trxFee, numberOfTrxs))
	test.CheckContractBalance(totalBalance)

	total_supply := big.NewInt(0)
	for _, balance := range cfg.GenesisBalances {
		total_supply.Add(total_supply, balance)
	}
	total_supply_uin256, _ := uint256.FromBig(total_supply)
	total_stake_uin256, _ := uint256.FromBig(total_stake)

	// Expected block reward
	var yield_curve dpos.YieldCurve
	yield_curve.Init(cfg)
	expected_block_reward_uint256, _ := yield_curve.CalculateBlockReward(total_stake_uin256, total_supply_uin256)
	expected_block_reward := expected_block_reward_uint256.ToBig()

	// Splitting block rewards between votes and blocks
	expected_dag_reward := bigutil.Div(bigutil.Mul(expected_block_reward, big.NewInt(int64(test.Chain_cfg.DPOS.DagProposersReward))), big.NewInt(100))
	expected_vote_reward := bigutil.Sub(expected_block_reward, expected_dag_reward)

	// Vote bonus rewards - aka Author reward
	maxBlockAuthorReward := big.NewInt(int64(DefaultChainCfg.DPOS.MaxBlockAuthorReward))
	bonus_reward := bigutil.Div(bigutil.Mul(expected_block_reward, maxBlockAuthorReward), big.NewInt(100))
	expected_vote_reward = bigutil.Sub(expected_vote_reward, bonus_reward)

	// Vote bonus rewards - aka Author reward
	max_votes_weigh := dpos.Max(tmp_rewards_stats.MaxVotesWeight, tmp_rewards_stats.TotalVotesWeight)
	threshold := max_votes_weigh*2/3 + 1
	author_reward := bigutil.Div(bigutil.Mul(bonus_reward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight-threshold))), big.NewInt(int64(max_votes_weigh-threshold)))

	// Expected participants rewards
	// validator1_rewards = (validator1_trxs * blockReward) / total_trxs
	validator1_total_reward := bigutil.Div(bigutil.Mul(expected_dag_reward, big.NewInt(int64(validator1_stats.DagBlocksCount))), big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount)))
	// Add vote reward
	validatorVoteReward := bigutil.Mul(big.NewInt(int64(validator1_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	validator1_total_reward = bigutil.Add(validator1_total_reward, validatorVoteReward)
	// Commission reward
	expected_validator1_commission_reward := bigutil.Div(bigutil.Mul(validator1_total_reward, big.NewInt(int64(validator1_commission))), big.NewInt(10000))
	expected_validator1_delegators_reward := bigutil.Sub(validator1_total_reward, expected_validator1_commission_reward)

	// Fee rewards goes to commission pool
	expected_validator1_commission_reward = bigutil.Add(expected_validator1_commission_reward, bigutil.Mul(trxFee, big.NewInt(int64(validator1_stats.DagBlocksCount))))

	// Add author reward
	author_commission_reward := bigutil.Div(bigutil.Mul(author_reward, big.NewInt(int64(validator1_commission))), big.NewInt(10000))
	author_reward = bigutil.Sub(author_reward, author_commission_reward)
	expected_validator1_delegators_reward = bigutil.Add(expected_validator1_delegators_reward, author_reward)
	expected_validator1_commission_reward = bigutil.Add(expected_validator1_commission_reward, author_commission_reward)

	// validator2_rewards = (validator2_trxs * blockReward) / total_trxs
	validator2_total_reward := bigutil.Div(bigutil.Mul(expected_dag_reward, big.NewInt(int64(validator2_stats.DagBlocksCount))), big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount)))
	// Add vote reward
	validatorVoteReward = bigutil.Mul(big.NewInt(int64(validator2_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	validator2_total_reward = bigutil.Add(validator2_total_reward, validatorVoteReward)

	expected_validator2_commission_reward := bigutil.Div(bigutil.Mul(validator2_total_reward, big.NewInt(int64(validator2_commission))), big.NewInt(10000))
	expected_validator2_delegators_reward := bigutil.Sub(validator2_total_reward, expected_validator2_commission_reward)

	// Fee rewards goes to commission pool
	expected_validator2_commission_reward = bigutil.Add(expected_validator2_commission_reward, bigutil.Mul(trxFee, big.NewInt(int64(validator2_stats.DagBlocksCount))))

	// Add vote reward for validator 4
	validatorVoteReward = bigutil.Mul(big.NewInt(int64(validator4_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	expected_delegator4_reward := validatorVoteReward

	// delegator 1(validator 1) gets 100 % from validator1_rewards
	expected_delegator1_reward := expected_validator1_delegators_reward

	// delegator 2(validator 2) gets 20 % from validator2_rewards
	expected_delegator2_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(20)), big.NewInt(100))

	// delegator 3 gets 80 % from validator2_rewards
	expected_delegator3_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(80)), big.NewInt(100))

	// expected_dag_rewardPlusFees := bigutil.Add(expected_dag_reward, bigutil.Mul(trxFee, big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount))))
	// expectedDelegatorsRewards := bigutil.Add(expected_delegator1_reward, bigutil.Add(expected_delegator2_reward, expected_delegator3_reward))
	// // Last digit is removed due to rounding error that makes these values unequal
	// tc.Assert.Equal(bigutil.Div(expected_dag_rewardPlusFees, big.NewInt(1)0), bigutil.Div(expectedDelegatorsRewards, big.NewInt(1)0))

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("claimRewards", validator2_addr), dpos.ErrNonExistentDelegation, util.ErrorString(""))

	// Check delgators rewards
	delegator1_old_balance := test.GetBalance(&delegator1_addr)
	delegator2_old_balance := test.GetBalance(&delegator2_addr)
	delegator3_old_balance := test.GetBalance(&delegator3_addr)
	delegator4_old_balance := test.GetBalance(&delegator4_addr)
	delegator4_old_balance.Sub(delegator4_old_balance, DefaultMinimumDeposit)

	// Check getter
	batch0_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getDelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetDelegationsRet)
	test.Unpack(batch0_parsed_result, "getDelegations", batch0_result.CodeRetval)
	tc.Assert.Equal(1, len(batch0_parsed_result.Delegations))
	tc.Assert.Equal(true, batch0_parsed_result.End)
	// Claims
	test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("claimRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, big.NewInt(0), test.Pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	{
		test.ExecuteAndCheck(delegator3_addr, big.NewInt(0), test.Pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
		clam_res := test.ExecuteAndCheck(delegator4_addr, DefaultMinimumDeposit, test.Pack("delegate", validator4_addr), util.ErrorString(""), util.ErrorString(""))
		tc.Assert.Equal(len(clam_res.Logs), 2)
		tc.Assert.Equal(clam_res.Logs[0].Topics[0], RewardsClaimedEventHash)
		tc.Assert.Equal(clam_res.Logs[1].Topics[0], DelegatedEventHash)
	}

	actual_delegator1_reward := bigutil.Sub(test.GetBalance(&delegator1_addr), delegator1_old_balance)
	actual_delegator2_reward := bigutil.Sub(test.GetBalance(&delegator2_addr), delegator2_old_balance)
	actual_delegator3_reward := bigutil.Sub(test.GetBalance(&delegator3_addr), delegator3_old_balance)
	actual_delegator4_reward := bigutil.Sub(test.GetBalance(&delegator4_addr), delegator4_old_balance)

	//Check claim vs getter result
	tc.Assert.Equal(batch0_parsed_result.Delegations[0].Delegation.Rewards, actual_delegator1_reward)

	tc.Assert.Equal(expected_delegator1_reward, actual_delegator1_reward)
	tc.Assert.Equal(expected_delegator2_reward, actual_delegator2_reward)
	tc.Assert.Equal(expected_delegator3_reward, actual_delegator3_reward)
	tc.Assert.Equal(expected_delegator4_reward, actual_delegator4_reward)

	// Check commission rewards
	validator1_old_balance := test.GetBalance(&validator1_owner)
	validator2_old_balance := test.GetBalance(&validator2_owner)
	validator4_old_balance := test.GetBalance(&validator4_owner)

	test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("claimCommissionRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, big.NewInt(0), test.Pack("claimCommissionRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	{
		claim_res := test.ExecuteAndCheck(delegator4_addr, big.NewInt(0), test.Pack("claimCommissionRewards", validator4_addr), util.ErrorString(""), util.ErrorString(""))
		tc.Assert.Equal(len(claim_res.Logs), 1)
		tc.Assert.Equal(claim_res.Logs[0].Topics[0], CommissionRewardsClaimedEventHash)
	}

	actual_validator1_commission_reward := bigutil.Sub(test.GetBalance(&validator1_owner), validator1_old_balance)
	actual_validator2_commission_reward := bigutil.Sub(test.GetBalance(&validator2_owner), validator2_old_balance)
	actual_validator4_commission_reward := bigutil.Sub(test.GetBalance(&validator4_owner), validator4_old_balance)

	tc.Assert.Equal(expected_validator1_commission_reward, actual_validator1_commission_reward)
	tc.Assert.Equal(expected_validator2_commission_reward, actual_validator2_commission_reward)
	tc.Assert.Equal(big.NewInt(0).Cmp(actual_validator4_commission_reward), 0)
	contractBalance := test.GetBalance(dpos.ContractAddress())
	if contractBalance.Cmp(total_stake) == -1 {
		t.Errorf("Balance left %d expected: %d", contractBalance, total_stake)
	}
}

func TestClaimAllRewards(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.DPOS.MinimumDeposit = big.NewInt(0)
	cfg.Hardforks.AspenHf.BlockNumPartTwo = 1000

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)

	total_stake := big.NewInt(0)

	// Create single delegator
	delegator_addr := addr(1)
	delegator_stake := DefaultMinimumDeposit

	validators_count := uint64(12)
	validator_stake := big.NewInt(0)
	validator_commission := uint16(0) // 0%
	var block_author common.Address

	var tmp_rewards_stats rewards_stats.RewardsStats
	// Add 1 extra validator, who is going to be block author with zero delegation
	for idx := uint64(1); idx <= validators_count+1; idx++ {
		validator_addr, validator_proof := generateAddrAndProof()
		validator_owner := addr(idx)
		test.ExecuteAndCheck(validator_owner, validator_stake, test.Pack("registerValidator", validator_addr, validator_proof, DefaultVrfKey, validator_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
		if idx == 1 {
			block_author = validator_addr
			tmp_rewards_stats = NewRewardsStats(&block_author)
			continue
		}

		// Delegate to each new validator
		test.ExecuteAndCheck(delegator_addr, delegator_stake, test.Pack("delegate", validator_addr), util.ErrorString(""), util.ErrorString(""))
		total_stake = bigutil.Add(total_stake, delegator_stake)

		// Create simulated rewards statistics for each validator
		validator_stats := rewards_stats.ValidatorStats{}
		validator_stats.DagBlocksCount = 1
		validator_stats.VoteWeight = 1
		tmp_rewards_stats.ValidatorsStats[validator_addr] = validator_stats

		tmp_rewards_stats.TotalDagBlocksCount += validator_stats.DagBlocksCount
		tmp_rewards_stats.TotalVotesWeight += validator_stats.VoteWeight
		tmp_rewards_stats.MaxVotesWeight += validator_stats.VoteWeight
	}

	// Advance block
	test.AdvanceBlock(&block_author, &tmp_rewards_stats)

	// Claim delegator's all rewards for batch 0
	claim_all_rewards_batch0_result := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("claimAllRewards"), util.ErrorString(""), util.ErrorString(""))

	tc.Assert.Equal(len(claim_all_rewards_batch0_result.Logs), int(validators_count))
	for log_idx := 0; log_idx < len(claim_all_rewards_batch0_result.Logs); log_idx++ {
		tc.Assert.Equal(claim_all_rewards_batch0_result.Logs[log_idx].Topics[0], RewardsClaimedEventHash)
	}
}

func TestGenesis(t *testing.T) {
	cfg := DefaultChainCfg

	delegator := addr(1)

	for i := uint64(1); i < 5; i++ {
		entry := chain_config.GenesisValidator{Address: addr(i), Owner: addr(i), VrfKey: DefaultVrfKey, Commission: 0, Endpoint: "", Description: "", Delegations: core.BalanceMap{}}
		entry.Delegations[delegator] = DefaultEligibilityBalanceThreshold
		cfg.DPOS.InitialValidators = append(cfg.DPOS.InitialValidators, entry)
	}
	accVoteCount := bigutil.Div(DefaultEligibilityBalanceThreshold, cfg.DPOS.VoteEligibilityBalanceStep)

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)

	defer test.End()

	totalAmountDelegated := bigutil.Mul(DefaultEligibilityBalanceThreshold, big.NewInt(4))
	test.CheckContractBalance(totalAmountDelegated)

	tc.Assert.Equal(bigutil.Sub(DefaultBalance, totalAmountDelegated), test.GetBalance(&delegator))
	tc.Assert.Equal(accVoteCount.Uint64()*4, test.GetDPOSReader().TotalEligibleVoteCount())
	tc.Assert.Equal(totalAmountDelegated, test.GetDPOSReader().TotalAmountDelegated())
	tc.Assert.Equal(accVoteCount.Uint64(), test.GetDPOSReader().GetEligibleVoteCount(addr_p(1)))
	tc.Assert.Equal(accVoteCount.Uint64(), test.GetDPOSReader().GetEligibleVoteCount(addr_p(2)))
	tc.Assert.Equal(accVoteCount.Uint64(), test.GetDPOSReader().GetEligibleVoteCount(addr_p(3)))
}

func TestSetValidatorInfo(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))

	validator_raw := test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal("test_description", validator.ValidatorInfo.Description)
	tc.Assert.Equal("test_endpoint", validator.ValidatorInfo.Endpoint)

	// Change description & endpoint and see it getValidator returns changed values
	{
		set_res := test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setValidatorInfo", val_addr, "modified_description", "modified_endpoint"), util.ErrorString(""), util.ErrorString(""))
		tc.Assert.Equal(len(set_res.Logs), 1)
		tc.Assert.Equal(set_res.Logs[0].Topics[0], ValidatorInfoSetEventHash)
	}
	validator_raw = test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal("modified_description", validator.ValidatorInfo.Description)
	tc.Assert.Equal("modified_endpoint", validator.ValidatorInfo.Endpoint)

	// Try to set invalid(too long) description & endpoint
	invalid_description := "100+char_description................................................................................."
	tc.Assert.Greater(len(invalid_description), dpos.MaxDescriptionLength)
	// ErrMaxDescriptionLengthExceeded
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setValidatorInfo", addr(2), invalid_description, "modified_endpoint"), dpos.ErrMaxDescriptionLengthExceeded, util.ErrorString(""))

	invalid_endpoint := "100+char_endpoint.................................."
	tc.Assert.Greater(len(invalid_endpoint), dpos.MaxEndpointLength)
	// ErrMaxEndpointLengthExceeded
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setValidatorInfo", addr(2), "modified_description", invalid_endpoint), dpos.ErrMaxEndpointLengthExceeded, util.ErrorString(""))

	// ErrWrongOwnerAcc
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setValidatorInfo", addr(2), "modified_description", "modified_endpoint"), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
}

func TestSetCommission(t *testing.T) {
	cfg := DefaultChainCfg
	cfg.DPOS.CommissionChangeDelta = 5
	cfg.DPOS.CommissionChangeFrequency = 4

	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), big.NewInt(0), test.Pack("setCommission", val_addr, uint16(11)), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setCommission", val_addr, uint16(11)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))

	//Advance 4 rounds
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setCommission", val_addr, uint16(dpos.MaxCommission+1)), dpos.ErrCommissionOverflow, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setCommission", val_addr, uint16(11)), util.ErrorString(""), util.ErrorString(""))

	//Advance 4 rounds
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setCommission", val_addr, uint16(20)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("setCommission", val_addr, uint16(16)), util.ErrorString(""), util.ErrorString(""))
}

func TestGetValidators(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 3*dpos.GetValidatorsMaxCount - 1

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := DefaultChainCfg
	validator_balance := bigutil.Mul(big.NewInt(100000000), TaraPrecision)
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = validator_balance
	}
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21400
	intristic_gas_batch1 := 21464
	intristic_gas_batch2 := 21464
	intristic_gas_batch3 := 21464

	// Get first batch of validators from contract
	batch0_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidators", uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch0_parsed_result, "getValidators", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetValidatorsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount, len(batch0_parsed_result.Validators))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(dpos.GetValidatorsMaxCount)+"_description", batch0_parsed_result.Validators[len(batch0_parsed_result.Validators)-1].Info.Description)

	// Get second batch of validators from contract
	batch1_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidators", uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch1_parsed_result, "getValidators", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetValidatorsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount, len(batch1_parsed_result.Validators))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(2*dpos.GetValidatorsMaxCount)+"_description", batch1_parsed_result.Validators[len(batch1_parsed_result.Validators)-1].Info.Description)

	// Get third batch of validators from contract
	batch2_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidators", uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch2_parsed_result, "getValidators", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetValidatorsMaxCount-1)+uint64(intristic_gas_batch2), batch2_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount - 1
	tc.Assert.Equal(dpos.GetValidatorsMaxCount-1, len(batch2_parsed_result.Validators))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(3*dpos.GetValidatorsMaxCount-1)+"_description", batch2_parsed_result.Validators[len(batch2_parsed_result.Validators)-1].Info.Description)

	// Get fourth batch of validators from contract - it should return 0 validators
	batch3_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidators", uint32(3) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch3_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch3_parsed_result, "getValidators", batch3_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas+uint64(intristic_gas_batch3), batch3_result.GasUsed)
	// Checks if number of returned validators is == 0
	tc.Assert.Equal(0, len(batch3_parsed_result.Validators))
	tc.Assert.Equal(true, batch3_parsed_result.End)
}

func TestGetValidatorsFor(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	val_owner := addr(uint64(1000))
	gen_validators_num := 3*dpos.GetValidatorsMaxCount - 1
	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := DefaultChainCfg
	validator_balance := bigutil.Mul(big.NewInt(100000000), TaraPrecision)
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = validator_balance
	}
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21656
	intristic_gas_batch1 := 21720

	// Get first batch of validators from contract
	batch0_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidatorsFor", val_owner, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch0_parsed_result, "getValidatorsFor", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetValidatorsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount, len(batch0_parsed_result.Validators))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(dpos.GetValidatorsMaxCount)+"_description", batch0_parsed_result.Validators[len(batch0_parsed_result.Validators)-1].Info.Description)

	// Get first batch of validators from contract
	batch1_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidatorsFor", val_owner, uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch1_parsed_result, "getValidatorsFor", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetValidatorsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount, len(batch1_parsed_result.Validators))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(2*dpos.GetValidatorsMaxCount)+"_description", batch1_parsed_result.Validators[len(batch1_parsed_result.Validators)-1].Info.Description)

	// Get first batch of validators from contract
	batch2_result := test.ExecuteAndCheck(addr(1), big.NewInt(0), test.Pack("getValidatorsFor", val_owner, uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetValidatorsRet)
	test.Unpack(batch2_parsed_result, "getValidatorsFor", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetValidatorsMaxCount)+uint64(intristic_gas_batch1), batch2_result.GasUsed)
	// Checks if number of returned validators is == dpos.GetValidatorsMaxCount
	tc.Assert.Equal(dpos.GetValidatorsMaxCount-1, len(batch2_parsed_result.Validators))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned validator in this batch is the right one based on description with idx of validator
	tc.Assert.Equal("validator_"+fmt.Sprint(3*dpos.GetValidatorsMaxCount-1)+"_description", batch2_parsed_result.Validators[len(batch2_parsed_result.Validators)-1].Info.Description)
}

func TestGetTotalDelegation(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 13

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := DefaultChainCfg
	validator_balance := bigutil.Mul(big.NewInt(100000000), TaraPrecision)
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = validator_balance
	}

	// Generate delegator and set some balance to him
	delegator1_addr := addr(uint64(gen_validators_num + 1))
	cfg.GenesisBalances[delegator1_addr] = validator_balance

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator delegations
	for i := 0; i < gen_validators_num; i++ {
		test.ExecuteAndCheck(delegator1_addr, DefaultMinimumDeposit, test.Pack("delegate", gen_validators[i].address), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas := 21464

	// Get first batch of delegator1 delegations from contract
	result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getTotalDelegation", delegator1_addr), util.ErrorString(""), util.ErrorString(""))
	parsed_result := new(GetTotalDelegationRet)
	test.Unpack(parsed_result, "getTotalDelegation", result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*uint64(gen_validators_num)+uint64(intristic_gas), result.GasUsed)
	// Checks total delegation
	tc.Assert.Equal(bigutil.Mul(big.NewInt(int64(gen_validators_num)), DefaultMinimumDeposit), parsed_result.TotalDelegation)

}

func TestGetDelegations(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 3 * dpos.GetDelegationsMaxCount
	gen_delegator1_delegations := gen_validators_num - 1

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := DefaultChainCfg
	validator_balance := bigutil.Mul(big.NewInt(100000000), TaraPrecision)
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = validator_balance
	}

	// Generate  delegator and set some balance to him
	delegator1_addr := addr(uint64(gen_validators_num + 1))
	cfg.GenesisBalances[delegator1_addr] = validator_balance

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator delegations
	for i := 0; i < gen_delegator1_delegations; i++ {
		test.ExecuteAndCheck(delegator1_addr, DefaultMinimumDeposit, test.Pack("delegate", gen_validators[i].address), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21592
	intristic_gas_batch1 := 21656
	intristic_gas_batch2 := 21656
	intristic_gas_batch3 := 21656

	// Get first batch of delegator1 delegations from contract
	batch0_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getDelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetDelegationsRet)
	test.Unpack(batch0_parsed_result, "getDelegations", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetDelegationsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned delegations is == dpos.GetDelegationsMaxCount
	tc.Assert.Equal(dpos.GetDelegationsMaxCount, len(batch0_parsed_result.Delegations))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned delegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[len(batch0_parsed_result.Delegations)-1].address, batch0_parsed_result.Delegations[len(batch0_parsed_result.Delegations)-1].Account)

	// Get second batch of delegator1 delegations from contract
	batch1_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getDelegations", delegator1_addr, uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetDelegationsRet)
	test.Unpack(batch1_parsed_result, "getDelegations", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetDelegationsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned delegations is == dpos.GetDelegationsMaxCount
	tc.Assert.Equal(dpos.GetDelegationsMaxCount, len(batch1_parsed_result.Delegations))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned delegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[dpos.GetDelegationsMaxCount+len(batch1_parsed_result.Delegations)-1].address, batch1_parsed_result.Delegations[len(batch1_parsed_result.Delegations)-1].Account)

	// Get third batch of delegator1 delegations from contract
	batch2_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getDelegations", delegator1_addr, uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetDelegationsRet)
	test.Unpack(batch2_parsed_result, "getDelegations", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetDelegationsMaxCount-1)+uint64(intristic_gas_batch2), batch2_result.GasUsed)
	// Checks if number of returned v is == dpos.GetDelegationsMaxCount - 1
	tc.Assert.Equal(dpos.GetDelegationsMaxCount-1, len(batch2_parsed_result.Delegations))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned delegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[2*dpos.GetDelegationsMaxCount+len(batch2_parsed_result.Delegations)-1].address, batch2_parsed_result.Delegations[len(batch2_parsed_result.Delegations)-1].Account)

	// Get fourth batch of delegator1 delegations from contract - it should return 0 delegations
	batch3_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getDelegations", delegator1_addr, uint32(3) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch3_parsed_result := new(GetDelegationsRet)
	test.Unpack(batch3_parsed_result, "getDelegations", batch3_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas+uint64(intristic_gas_batch3), batch3_result.GasUsed)
	// Checks if number of returned delegations is == 0
	tc.Assert.Equal(0, len(batch3_parsed_result.Delegations))
	tc.Assert.Equal(true, batch3_parsed_result.End)
}

func TestGetUndelegationsV1(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 3 * dpos.GetUndelegationsMaxCount
	gen_delegator1_delegations := gen_validators_num - 1

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := DefaultChainCfg
	validator_balance := bigutil.Mul(big.NewInt(100000000), TaraPrecision)
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = validator_balance
	}

	cfg.Hardforks.MagnoliaHf.BlockNum = 1000

	// Generate 2 delegators and set some balance to them
	delegator1_addr := addr(uint64(gen_validators_num + 1))
	cfg.GenesisBalances[delegator1_addr] = validator_balance

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	// Register validators
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator delegations
	for i := 0; i < gen_delegator1_delegations; i++ {
		test.ExecuteAndCheck(delegator1_addr, DefaultMinimumDeposit, test.Pack("delegate", gen_validators[i].address), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator undelegations
	for i := 0; i < gen_delegator1_delegations; i++ {
		test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("undelegate", gen_validators[i].address, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	}

	intristic_gas_batch0 := 21592
	intristic_gas_batch1 := 21656
	intristic_gas_batch2 := 21656
	intristic_gas_batch3 := 21656

	// Get first batch of delegator1 undelegations from contract
	batch0_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetUndelegationsRet)
	test.Unpack(batch0_parsed_result, "getUndelegations", batch0_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetUndelegationsMaxCount+uint64(intristic_gas_batch0), batch0_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount, len(batch0_parsed_result.Undelegations))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[len(batch0_parsed_result.Undelegations)-1].address, batch0_parsed_result.Undelegations[len(batch0_parsed_result.Undelegations)-1].Validator)

	// Get second batch of delegator1 undelegations from contract
	batch1_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegations", delegator1_addr, uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetUndelegationsRet)
	test.Unpack(batch1_parsed_result, "getUndelegations", batch1_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*dpos.GetUndelegationsMaxCount+uint64(intristic_gas_batch1), batch1_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount, len(batch1_parsed_result.Undelegations))
	tc.Assert.Equal(false, batch1_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[dpos.GetUndelegationsMaxCount+len(batch1_parsed_result.Undelegations)-1].address, batch1_parsed_result.Undelegations[len(batch1_parsed_result.Undelegations)-1].Validator)

	// Get third batch of delegator1 undelegations from contract
	batch2_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegations", delegator1_addr, uint32(2) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch2_parsed_result := new(GetUndelegationsRet)
	test.Unpack(batch2_parsed_result, "getUndelegations", batch2_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas*(dpos.GetUndelegationsMaxCount-1)+uint64(intristic_gas_batch2), batch2_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount - 1
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount-1, len(batch2_parsed_result.Undelegations))
	tc.Assert.Equal(true, batch2_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	tc.Assert.Equal(gen_validators[2*dpos.GetUndelegationsMaxCount+len(batch2_parsed_result.Undelegations)-1].address, batch2_parsed_result.Undelegations[len(batch2_parsed_result.Undelegations)-1].Validator)

	// Get fourth batch of delegator1 undelegations from contract - it should return 0 undelegations
	batch3_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegations", delegator1_addr, uint32(3) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch3_parsed_result := new(GetUndelegationsRet)
	test.Unpack(batch3_parsed_result, "getUndelegations", batch3_result.CodeRetval)
	// Checks used gas
	tc.Assert.Equal(dpos.DposBatchGetMethodsGas+uint64(intristic_gas_batch3), batch3_result.GasUsed)
	// Checks if number of returned undelegations is == 0
	tc.Assert.Equal(0, len(batch3_parsed_result.Undelegations))
	tc.Assert.Equal(true, batch3_parsed_result.End)

	// Test getUndelegations after all of the delegators undelegate from validator
	undelegations1_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	undelegations1_parsed_result := new(GetUndelegationsRet)
	test.Unpack(undelegations1_parsed_result, "getUndelegations", undelegations1_result.CodeRetval)
	tc.Assert.Equal(false, undelegations1_parsed_result.End)
	tc.Assert.Equal(true, undelegations1_parsed_result.Undelegations[0].ValidatorExists)
	// Last delegator undelegates from gen_validators[0].address
	test.ExecuteAndCheck(gen_validators[0].owner, big.NewInt(0), test.Pack("undelegate", gen_validators[0].address, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	undelegations2_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	undelegations2_parsed_result := new(GetUndelegationsRet)
	test.Unpack(undelegations2_parsed_result, "getUndelegations", undelegations2_result.CodeRetval)
	tc.Assert.Equal(false, undelegations2_parsed_result.End)
	tc.Assert.Equal(len(undelegations1_parsed_result.Undelegations), len(undelegations2_parsed_result.Undelegations))
	tc.Assert.Equal(false, undelegations2_parsed_result.Undelegations[0].ValidatorExists)
}

func TestGetUndelegationsV2(t *testing.T) {
	type GenValidator struct {
		address common.Address
		proof   []byte
		owner   common.Address
	}

	gen_validators_num := 4

	// Generate gen_validators_num validators
	var gen_validators []GenValidator
	for i := 1; i <= gen_validators_num; i++ {
		val_addr, val_proof := generateAddrAndProof()
		val_owner := addr(uint64(i))

		gen_validators = append(gen_validators, GenValidator{val_addr, val_proof, val_owner})
	}

	// Set some balance to validators
	cfg := DefaultChainCfg
	validator_balance := bigutil.Mul(big.NewInt(100000000), TaraPrecision)
	for _, validator := range gen_validators {
		cfg.GenesisBalances[validator.owner] = validator_balance
	}

	cfg.Hardforks.MagnoliaHf.BlockNum = 1000
	cfg.Hardforks.CornusHf.BlockNum = 0

	// Create delegator with initial balance
	delegator1_addr := addr(uint64(gen_validators_num + 1))
	cfg.GenesisBalances[delegator1_addr] = validator_balance

	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	// Register validators and delegate to them
	for idx, validator := range gen_validators {
		test.ExecuteAndCheck(validator.owner, DefaultMinimumDeposit, test.Pack("registerValidator", validator.address, validator.proof, DefaultVrfKey, uint16(10), "validator_"+fmt.Sprint(idx+1)+"_description", "test_endpoint"), util.ErrorString(""), util.ErrorString(""))
		test.ExecuteAndCheck(delegator1_addr, DefaultEligibilityBalanceThreshold, test.Pack("delegate", validator.address), util.ErrorString(""), util.ErrorString(""))
	}

	// Create delegator undelegations
	undelegations_count := 0
	for validator_idx, validator := range gen_validators {
		// Gen multiple undelegations
		for undelegation_idx := 0; undelegation_idx < (validator_idx+1)*3; undelegation_idx++ {
			test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("undelegateV2", validator.address, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
			undelegations_count++
		}
	}

	intristic_gas_batch0 := 21592
	intristic_gas_batch1 := 21656

	// Get first batch of delegator1 undelegations from contract
	batch0_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegationsV2", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetUndelegationsV2Ret)
	test.Unpack(batch0_parsed_result, "getUndelegationsV2", batch0_result.CodeRetval)
	// Checks used gas
	batch0_expected_gas := (8+2*dpos.GetUndelegationsMaxCount)*dpos.DposBatchGetMethodsGas + uint64(intristic_gas_batch0)
	tc.Assert.Equal(batch0_expected_gas, batch0_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount
	tc.Assert.Equal(dpos.GetUndelegationsMaxCount, len(batch0_parsed_result.UndelegationsV2))
	tc.Assert.Equal(false, batch0_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	for undelegation_idx, undelegation := range batch0_parsed_result.UndelegationsV2 {
		var validator common.Address
		if undelegation_idx < 3 {
			validator = gen_validators[0].address
		} else if undelegation_idx < 9 {
			validator = gen_validators[1].address
		} else if undelegation_idx < 18 {
			validator = gen_validators[2].address
		} else {
			validator = gen_validators[3].address
		}

		tc.Assert.Equal(validator, undelegation.UndelegationData.Validator)
	}

	// Get second batch of delegator1 undelegations from contract
	batch1_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getUndelegationsV2", delegator1_addr, uint32(1) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch1_parsed_result := new(GetUndelegationsV2Ret)
	test.Unpack(batch1_parsed_result, "getUndelegationsV2", batch1_result.CodeRetval)
	// Checks used gas
	batch1_expected_gas := (8+2*10)*dpos.DposBatchGetMethodsGas + uint64(intristic_gas_batch1)
	tc.Assert.Equal(batch1_expected_gas, batch1_result.GasUsed)
	// Checks if number of returned undelegations is == dpos.GetUndelegationsMaxCount
	tc.Assert.Equal(undelegations_count-dpos.GetUndelegationsMaxCount, len(batch1_parsed_result.UndelegationsV2))
	tc.Assert.Equal(true, batch1_parsed_result.End)
	// Checks if last returned undelegation in this batch is the right one based on validator address
	for _, undelegation := range batch1_parsed_result.UndelegationsV2 {
		tc.Assert.Equal(gen_validators[3].address, undelegation.UndelegationData.Validator)
	}
}

func TestGetValidator(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	// ErrNonExistentValidator
	test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))

	// Register validator and check if it is returned from contract
	test.ExecuteAndCheck(val_owner, DefaultMinimumDeposit, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	validator_raw := test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("getValidator", val_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	tc.Assert.Equal(DefaultMinimumDeposit, validator.ValidatorInfo.TotalStake)
	tc.Assert.Equal(val_owner, validator.ValidatorInfo.Owner)

	// Undelegate
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("undelegate", val_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))
	// Advance 3 more rounds - delegation locking periods == 4
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)
	test.ExecuteAndCheck(val_owner, big.NewInt(0), test.Pack("confirmUndelegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// ErrNonExistentValidator
	test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("getValidator", val_addr), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestGetTotalEligibleVotesCount(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	delegator_addr := addr(2)

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks, otherwise getters are not working properly - in case these is not enough blocks produced yet, getters are not delayed as they should be
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	// Register validator and see what is getTotalEligibleVotesCount
	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// New delegation through registerValidator should not be applied yet in delayed storage - getTotalEligibleVotesCount should return 0 at this moment
	votes_count_raw := test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count := new(uint64)
	test.Unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)
	tc.Assert.Equal(uint64(0), *votes_count)

	// Wait DelegationDelay so getTotalEligibleVotesCount returns votes count based on new delegation
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}
	votes_count_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count = new(uint64)
	test.Unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)

	expected_votes_count := bigutil.Div(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *votes_count)

	// Delegate and see what is getTotalEligibleVotesCount
	test.ExecuteAndCheck(delegator_addr, bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(2)), test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	// Wait DelegationDelay so getTotalEligibleVotesCount returns votes count based on new delegation
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}
	votes_count_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count = new(uint64)
	test.Unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)

	expected_votes_count = bigutil.Div(bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(3)), test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *votes_count)

	// Undelegate and see what is getTotalEligibleVotesCount
	test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("undelegate", val_addr, test.Chain_cfg.DPOS.EligibilityBalanceThreshold), util.ErrorString(""), util.ErrorString(""))
	// Wait DelegationDelay so getTotalEligibleVotesCount returns votes count based on new delegation
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}
	votes_count_raw = test.ExecuteAndCheck(delegator_addr, big.NewInt(0), test.Pack("getTotalEligibleVotesCount"), util.ErrorString(""), util.ErrorString(""))
	votes_count = new(uint64)
	test.Unpack(votes_count, "getTotalEligibleVotesCount", votes_count_raw.CodeRetval)

	expected_votes_count = bigutil.Div(bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(2)), test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *votes_count)
}

func TestGetValidatorEligibleVotesCount(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks, otherwise getters are not working properly - in case these is not enough blocks produced yet, getters are not delayed as they should be
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	// Register validator
	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Delegate some more
	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.Pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	// Wait DelegationDelay so new delegation is applied
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	// check if validator vote count was calculated properly in contract
	val_votes_count_raw := test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("getValidatorEligibleVotesCount", val_addr), util.ErrorString(""), util.ErrorString(""))
	val_votes_count := new(uint64)
	test.Unpack(val_votes_count, "getValidatorEligibleVotesCount", val_votes_count_raw.CodeRetval)

	expected_votes_count := bigutil.Div(bigutil.Mul(test.Chain_cfg.DPOS.EligibilityBalanceThreshold, big.NewInt(2)), test.Chain_cfg.DPOS.VoteEligibilityBalanceStep)
	tc.Assert.Equal(expected_votes_count.Uint64(), *val_votes_count)
}

func TestIsValidatorEligible(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	val_owner := addr(1)
	val_addr, proof := generateAddrAndProof()

	// Advance test.Chain_cfg.DPOS.DelegationDelay blocks, otherwise getters are not working properly - in case these is not enough blocks produced yet, getters are not delayed as they should be
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	// Check if validatorEligible == false before register&delegate
	is_eligible_raw := test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("isValidatorEligible", val_addr), util.ErrorString(""), util.ErrorString(""))
	is_eligible := new(bool)
	test.Unpack(is_eligible, "isValidatorEligible", is_eligible_raw.CodeRetval)
	tc.Assert.Equal(false, *is_eligible)

	test.ExecuteAndCheck(val_owner, test.Chain_cfg.DPOS.EligibilityBalanceThreshold, test.Pack("registerValidator", val_addr, proof, DefaultVrfKey, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))

	// Wait DelegationDelay so new delegation is applied
	for i := 0; i < int(test.Chain_cfg.DPOS.DelegationDelay); i++ {
		test.AdvanceBlock(nil, nil)
	}

	// Check if validatorEligible == true after register&delegate
	is_eligible_raw = test.ExecuteAndCheck(val_addr, big.NewInt(0), test.Pack("isValidatorEligible", val_addr), util.ErrorString(""), util.ErrorString(""))
	is_eligible = new(bool)
	test.Unpack(is_eligible, "isValidatorEligible", is_eligible_raw.CodeRetval)
	tc.Assert.Equal(true, *is_eligible)
}

func TestIterableMapClass(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.St.BeginBlock(&vm.BlockInfo{})

	var storage contract_storage.StorageWrapper
	dpos_contract_address := dpos.ContractAddress()
	storage.Init(dpos_contract_address, test.GetEvmStateStorage())

	iter_map_prefix := []byte{0}
	iter_map := contract_storage.AddressesIMap{}
	iter_map.Init(&storage, iter_map_prefix)

	acc1 := addr(1)
	acc2 := addr(2)
	acc3 := addr(3)
	acc4 := addr(4)

	// Tests CreateAccount & GetCount
	iter_map.CreateAccount(&acc1)
	tc.Assert.Equal(uint32(1), iter_map.GetCount())
	// Tries to create duplicate account
	tc.Assert.PanicsWithValue("Item "+string(acc1.Bytes())+" already exists", func() { iter_map.CreateAccount(&acc1) })
	tc.Assert.Equal(uint32(1), iter_map.GetCount())

	iter_map.CreateAccount(&acc2)
	iter_map.CreateAccount(&acc3)
	iter_map.CreateAccount(&acc4)
	tc.Assert.Equal(uint32(4), iter_map.GetCount())

	// Tests GetAccounts
	items_in_batch := uint32(2)
	batch0_accounts, end := iter_map.GetAccounts(0 /* batch 0 */, items_in_batch)
	tc.Assert.Equal(items_in_batch, uint32(len(batch0_accounts)))
	tc.Assert.Equal(acc1, batch0_accounts[0])
	tc.Assert.Equal(acc2, batch0_accounts[1])
	tc.Assert.Equal(false, end)

	batch1_accounts, end := iter_map.GetAccounts(1 /* batch 1 */, items_in_batch)
	tc.Assert.Equal(items_in_batch, uint32(len(batch1_accounts)))
	tc.Assert.Equal(acc3, batch1_accounts[0])
	tc.Assert.Equal(acc4, batch1_accounts[1])
	tc.Assert.Equal(true, end)

	// Tests RemoveAccount
	iter_map.RemoveAccount(&acc2)
	tc.Assert.Equal(uint32(3), iter_map.GetCount())
	tc.Assert.PanicsWithValue("Item "+string(acc2.Bytes())+" does not exist", func() { iter_map.RemoveAccount(&acc2) })
	tc.Assert.Equal(uint32(3), iter_map.GetCount())

	// To optimize iterbale map removing, it is implement through swapping of the item to be deleted with the last item
	// and then intenal array is just downsized by 1
	items_in_batch = uint32(3)
	accounts, end := iter_map.GetAccounts(0 /* batch 0 */, items_in_batch)
	tc.Assert.Equal(items_in_batch, uint32(len(accounts)))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(acc1, accounts[0])
	// acc2 was deleted, so acc4 should be now at acc2 original position(index)
	tc.Assert.Equal(acc4, accounts[1])
	tc.Assert.Equal(acc3, accounts[2])

	// Tests AccountExists
	tc.Assert.Equal(true, iter_map.AccountExists(&acc1))
	tc.Assert.Equal(false, iter_map.AccountExists(&acc2))
	tc.Assert.Equal(true, iter_map.AccountExists(&acc3))
	tc.Assert.Equal(true, iter_map.AccountExists(&acc4))
}

func TestValidatorsClass(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.St.BeginBlock(&vm.BlockInfo{})

	var storage contract_storage.StorageWrapper
	dpos_contract_address := dpos.ContractAddress()
	storage.Init(dpos_contract_address, test.GetEvmStateStorage())

	validators := new(dpos.Validators).Init(&storage, []byte{})
	field_validators := []byte{0}
	validators.Init(&storage, field_validators)

	validator1_addr, _ := generateAddrAndProof()
	validator1_owner := addr(1)

	validator2_addr, _ := generateAddrAndProof()
	validator2_owner := addr(1)

	// Checks CreateValidator & CheckValidatorOwner
	validators.CreateValidator(true, &validator1_owner, &validator1_addr, DefaultVrfKey, 0, 1, "validator1_description", "validator1_endpoint")
	validators.CheckValidatorOwner(&validator1_owner, &validator1_addr)
	tc.Assert.Equal(uint32(1), validators.GetValidatorsCount())

	validators.CreateValidator(true, &validator2_owner, &validator2_addr, DefaultVrfKey, 0, 2, "validator2_description", "validator2_endpoint")
	validators.CheckValidatorOwner(&validator2_owner, &validator2_addr)
	tc.Assert.Equal(uint32(2), validators.GetValidatorsCount())
	{
		// Checks GetValidator & GetValidatorInfo
		validator1 := validators.GetValidator(&validator1_addr)
		tc.Assert.Equal(uint16(1), validator1.Commission)
		validator1_info := validators.GetValidatorInfo(&validator1_addr)
		tc.Assert.Equal("validator1_description", validator1_info.Description)
		tc.Assert.Equal("validator1_endpoint", validator1_info.Endpoint)

		// Checks ModifyValidator & ModifyValidatorInfo
		validator1.Commission = 11
		validator1_info.Description = "validator1_description_modified"
		validator1_info.Endpoint = "validator1_endpoint_modified"
		validators.ModifyValidator(true, &validator1_addr, validator1)
		validators.ModifyValidatorInfo(&validator1_addr, validator1_info)
	}

	validator1 := validators.GetValidator(&validator1_addr)
	tc.Assert.Equal(uint16(11), validator1.Commission)
	validator1_info := validators.GetValidatorInfo(&validator1_addr)
	tc.Assert.Equal("validator1_description_modified", validator1_info.Description)
	tc.Assert.Equal("validator1_endpoint_modified", validator1_info.Endpoint)

	// Checks GetValidatorsAddresses
	validators_addresses, end := validators.GetValidatorsAddresses(uint32(0), uint32(2))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(2, len(validators_addresses))
	tc.Assert.Equal(validators.GetValidatorsCount(), uint32(len(validators_addresses)))
	tc.Assert.Equal(validator1_addr, validators_addresses[0])
	tc.Assert.Equal(validator2_addr, validators_addresses[1])

	// Checks DeleteValidator
	tc.Assert.Equal(true, validators.ValidatorExists(&validator1_addr))
	validators.DeleteValidator(&validator1_addr)
	tc.Assert.Equal(false, validators.ValidatorExists(&validator1_addr))
	tc.Assert.Equal(uint32(1), validators.GetValidatorsCount())

	validator3_addr := addr(3)
	tc.Assert.PanicsWithValue("Modify: non existent validator", func() { validators.ModifyValidator(true, &validator3_addr, validator1) })
	tc.Assert.PanicsWithValue("Modify: non existent validator", func() { validators.ModifyValidatorInfo(&validator3_addr, validator1_info) })
}

func TestDelegationsClass(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.St.BeginBlock(&vm.BlockInfo{})

	var storage contract_storage.StorageWrapper
	dpos_contract_address := dpos.ContractAddress()
	storage.Init(dpos_contract_address, test.GetEvmStateStorage())

	delegations := dpos.Delegations{}
	field_delegations := []byte{2}
	delegations.Init(&storage, field_delegations)

	validator1_addr := addr(1)
	validator2_addr := addr(2)

	delegator1_addr := addr(3)

	// Check getters to 0 values
	tc.Assert.Equal(false, delegations.DelegationExists(&delegator1_addr, &validator1_addr))
	tc.Assert.Equal(uint32(0), delegations.GetDelegationsCount(&delegator1_addr))

	delegations_ret, end := delegations.GetDelegatorValidatorsAddresses(&delegator1_addr, 0, 10)
	tc.Assert.Equal(0, len(delegations_ret))
	tc.Assert.Equal(true, end)

	delegation_ret := delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	var delegation_nil_ptr *dpos.Delegation = nil
	tc.Assert.Equal(delegation_nil_ptr, delegation_ret)

	// Creates 2 delegations
	delegations.CreateDelegation(&delegator1_addr, &validator1_addr, 0, big.NewInt(50))
	delegations.CreateDelegation(&delegator1_addr, &validator2_addr, 0, big.NewInt(50))

	// Check GetDelegationsCount + DelegationExists
	tc.Assert.Equal(uint32(2), delegations.GetDelegationsCount(&delegator1_addr))
	tc.Assert.Equal(true, delegations.DelegationExists(&delegator1_addr, &validator1_addr))

	// Check GetDelegatorValidatorsAddresses
	delegations_ret, end = delegations.GetDelegatorValidatorsAddresses(&delegator1_addr, 0, 10)
	tc.Assert.Equal(2, len(delegations_ret))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(validator1_addr, delegations_ret[0])
	tc.Assert.Equal(validator2_addr, delegations_ret[1])

	// Check GetDelegation
	delegation_ret = delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(uint64(0), delegation_ret.LastUpdated)
	tc.Assert.Equal(big.NewInt(50), delegation_ret.Stake)

	// Check ModifyDelegation
	delegation_ret.LastUpdated = 1
	delegation_ret.Stake = big.NewInt(10)
	delegations.ModifyDelegation(&delegator1_addr, &validator1_addr, delegation_ret)

	delegation_ret = delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(uint64(1), delegation_ret.LastUpdated)
	tc.Assert.Equal(big.NewInt(10), delegation_ret.Stake)

	// Check RemoveDelegation
	delegations.RemoveDelegation(&delegator1_addr, &validator1_addr)
	delegation_ret = delegations.GetDelegation(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(delegation_nil_ptr, delegation_ret)
	tc.Assert.Equal(uint32(1), delegations.GetDelegationsCount(&delegator1_addr))
	tc.Assert.Equal(false, delegations.DelegationExists(&delegator1_addr, &validator1_addr))
}

func TestUndelegationsClass(t *testing.T) {
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, CopyDefaultChainConfig())
	defer test.End()

	// Must be here to setup some internal data in evm_state, otherwise it is not possible to write into contract storage
	test.St.BeginBlock(&vm.BlockInfo{})

	var storage contract_storage.StorageWrapper
	dpos_contract_address := dpos.ContractAddress()
	storage.Init(dpos_contract_address, test.GetEvmStateStorage())

	undelegations := dpos.Undelegations{}
	field_undelegations := []byte{3}
	undelegations.Init(&storage, field_undelegations)

	validator1_addr := addr(1)
	validator2_addr := addr(2)

	delegator1_addr := addr(3)

	// Check getters to 0 values

	tc.Assert.Equal(false, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr, nil))
	tc.Assert.Equal(uint32(0), undelegations.GetUndelegationsV1Count(&delegator1_addr))
	tc.Assert.Equal(uint32(0), undelegations.GetUndelegationsV2Count(&delegator1_addr))

	undelegations_v1_validators_ret, end := undelegations.GetUndelegationsV1Validators(&delegator1_addr, 0, 10)
	tc.Assert.Equal(0, len(undelegations_v1_validators_ret))
	tc.Assert.Equal(true, end)

	var empty_address *common.Address
	undelegations_v2_validator_ret, end := undelegations.GetUndelegationsV2Validator(&delegator1_addr, 0)
	tc.Assert.Equal(empty_address, undelegations_v2_validator_ret)
	tc.Assert.Equal(true, end)

	undelegation_ret := undelegations.GetUndelegationBaseObject(&delegator1_addr, &validator1_addr, nil)
	var undelegation_nil_ptr *dpos.UndelegationV1 = nil
	tc.Assert.Equal(undelegation_nil_ptr, undelegation_ret)

	// Creates 2 undelegations - V1 and V2 (with undelegation_id)
	undelegations.CreateUndelegationV1(&delegator1_addr, &validator1_addr, 0, big.NewInt(50))
	undelegation_id := undelegations.CreateUndelegationV2(&delegator1_addr, &validator2_addr, 0, big.NewInt(100))

	// Check GetUndelegationsCount + UndelegationExists
	tc.Assert.Equal(uint32(1), undelegations.GetUndelegationsV1Count(&delegator1_addr))
	tc.Assert.Equal(uint32(1), undelegations.GetUndelegationsV2Count(&delegator1_addr))
	tc.Assert.Equal(true, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr, nil))
	tc.Assert.Equal(true, undelegations.UndelegationExists(&delegator1_addr, &validator2_addr, &undelegation_id))

	// Check GetUndelegationsValidators
	undelegations_v1_ret, end := undelegations.GetUndelegationsV1Validators(&delegator1_addr, 0, 10)
	tc.Assert.Equal(1, len(undelegations_v1_ret))
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(validator1_addr, undelegations_v1_ret[0])

	undelegations_v2_ret, end := undelegations.GetUndelegationsV2Validator(&delegator1_addr, 0)
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(&validator2_addr, undelegations_v2_ret)

	undelegations_v2_ret, end = undelegations.GetUndelegationsV2Validator(&delegator1_addr, 1)
	tc.Assert.Equal(true, end)
	tc.Assert.Equal(empty_address, undelegations_v2_ret)

	// Check GetUndelegation
	undelegation_ret = undelegations.GetUndelegationBaseObject(&delegator1_addr, &validator1_addr, nil)
	tc.Assert.Equal(uint64(0), undelegation_ret.Block)
	tc.Assert.Equal(big.NewInt(50), undelegation_ret.Amount)

	undelegation_ret = undelegations.GetUndelegationBaseObject(&delegator1_addr, &validator2_addr, &undelegation_id)
	tc.Assert.Equal(uint64(0), undelegation_ret.Block)
	tc.Assert.Equal(big.NewInt(100), undelegation_ret.Amount)

	undelegation_v1_ret := undelegations.GetUndelegationV1(&delegator1_addr, &validator1_addr)
	tc.Assert.Equal(uint64(0), undelegation_v1_ret.Block)
	tc.Assert.Equal(big.NewInt(50), undelegation_v1_ret.Amount)

	undelegation_v2_ret := undelegations.GetUndelegationV2(&delegator1_addr, &validator2_addr, undelegation_id)
	tc.Assert.Equal(uint64(0), undelegation_v2_ret.Block)
	tc.Assert.Equal(big.NewInt(100), undelegation_v2_ret.Amount)
	tc.Assert.Equal(undelegation_id, undelegation_v2_ret.Id)

	// Check RemoveDelegation
	undelegations.RemoveUndelegation(&delegator1_addr, &validator1_addr, nil)
	undelegation_ret = undelegations.GetUndelegationBaseObject(&delegator1_addr, &validator1_addr, nil)
	tc.Assert.Equal(undelegation_nil_ptr, undelegation_ret)
	tc.Assert.Equal(uint32(0), undelegations.GetUndelegationsV1Count(&delegator1_addr))
	tc.Assert.Equal(uint32(1), undelegations.GetUndelegationsV2Count(&delegator1_addr))
	tc.Assert.Equal(false, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr, nil))

	undelegations.RemoveUndelegation(&delegator1_addr, &validator2_addr, &undelegation_id)
	undelegation_ret = undelegations.GetUndelegationBaseObject(&delegator1_addr, &validator2_addr, &undelegation_id)
	tc.Assert.Equal(undelegation_nil_ptr, undelegation_ret)
	tc.Assert.Equal(uint32(0), undelegations.GetUndelegationsV1Count(&delegator1_addr))
	tc.Assert.Equal(uint32(0), undelegations.GetUndelegationsV2Count(&delegator1_addr))
	tc.Assert.Equal(false, undelegations.UndelegationExists(&delegator1_addr, &validator1_addr, &undelegation_id))
}

func TestMakeLogsCheckTopics(t *testing.T) {
	tc := tests.NewTestCtx(t)
	amount := big.NewInt(0)

	Abi, _ := abi.JSON(strings.NewReader(dpos_sol.TaraxaDposClientMetaData))
	logs := *new(dpos.Logs).Init(Abi.Events)

	undelegation_id := uint64(1)
	count := 0
	{
		log := logs.MakeDelegatedLog(&common.ZeroAddress, &common.ZeroAddress, amount)
		tc.Assert.Equal(log.Topics[0], DelegatedEventHash)
		count++
	}
	{
		log := logs.MakeUndelegatedV1Log(&common.ZeroAddress, &common.ZeroAddress, amount)
		tc.Assert.Equal(log.Topics[0], UndelegatedEventHash)
		count++
	}
	{
		log := logs.MakeUndelegatedV2Log(&common.ZeroAddress, &common.ZeroAddress, 1, amount)
		tc.Assert.Equal(log.Topics[0], UndelegatedV2EventHash)
		count++
	}
	{
		log := logs.MakeUndelegateConfirmedLog(&common.ZeroAddress, &common.ZeroAddress, nil, amount)
		tc.Assert.Equal(log.Topics[0], UndelegateConfirmedEventHash)
		count++
	}
	{
		log := logs.MakeUndelegateConfirmedLog(&common.ZeroAddress, &common.ZeroAddress, &undelegation_id, amount)
		tc.Assert.Equal(log.Topics[0], UndelegateConfirmedV2EventHash)
		count++
	}
	{
		log := logs.MakeUndelegateCanceledLog(&common.ZeroAddress, &common.ZeroAddress, nil, amount)
		tc.Assert.Equal(log.Topics[0], UndelegateCanceledEventHash)
		count++
	}
	{
		log := logs.MakeUndelegateCanceledLog(&common.ZeroAddress, &common.ZeroAddress, &undelegation_id, amount)
		tc.Assert.Equal(log.Topics[0], UndelegateCanceledV2EventHash)
		count++
	}
	{
		log := logs.MakeRedelegatedLog(&common.ZeroAddress, &common.ZeroAddress, &common.ZeroAddress, amount)
		tc.Assert.Equal(log.Topics[0], RedelegatedEventHash)
		count++
	}
	{
		log := logs.MakeRewardsClaimedLog(&common.ZeroAddress, &common.ZeroAddress, big.NewInt(111))
		tc.Assert.Equal(log.Topics[0], RewardsClaimedEventHash)
		count++
	}
	{
		log := logs.MakeCommissionRewardsClaimedLog(&common.ZeroAddress, &common.ZeroAddress, big.NewInt(222))
		tc.Assert.Equal(log.Topics[0], CommissionRewardsClaimedEventHash)
		count++
	}
	{
		log := logs.MakeCommissionSetLog(&common.ZeroAddress, 0)
		tc.Assert.Equal(log.Topics[0], CommissionSetEventHash)
		count++
	}
	{
		log := logs.MakeValidatorRegisteredLog(&common.ZeroAddress)
		tc.Assert.Equal(log.Topics[0], ValidatorRegisteredEventHash)
		count++
	}
	{
		log := logs.MakeValidatorInfoSetLog(&common.ZeroAddress)
		tc.Assert.Equal(log.Topics[0], ValidatorInfoSetEventHash)
		count++
	}
	// Check that we tested all events from the ABI
	tc.Assert.Equal(count, len(Abi.Events))
}

func TestRedelegateHF(t *testing.T) {
	trxFee := bigutil.Div(TaraPrecision, big.NewInt(1000)) //  0.001 TARA

	validator1_addr, validator1_proof := generateAddrAndProof()
	validator1_owner := addr(1)
	validator1_commission := uint16(500) // 5%
	delegator1_addr := validator1_owner
	delegator1_stake := DefaultMinimumDeposit

	validator2_addr, validator2_proof := generateAddrAndProof()
	validator2_owner := addr(2)
	validator2_commission := uint16(200) // 2%
	delegator2_addr := validator2_owner
	delegator2_stake := DefaultMinimumDeposit

	delegator3_addr := addr(3)
	delegator3_stake := bigutil.Mul(DefaultMinimumDeposit, big.NewInt(4))

	validator4_addr, validator4_proof := generateAddrAndProof()
	validator4_owner := addr(4)
	validator4_commission := uint16(0) // 0%
	delegator4_addr := validator4_owner
	delegator4_stake := DefaultMinimumDeposit

	validator5_addr, validator5_proof := generateAddrAndProof()
	validator5_owner := addr(5)
	validator5_commission := uint16(10000) //100%
	// delegator5_addr := validator5_owner
	delegator5_stake := DefaultMinimumDeposit

	cfg := CopyDefaultChainConfig()
	cfg.Hardforks.AspenHf.BlockNumPartTwo = 1000
	cfg.Hardforks.FixRedelegateBlockNum = 12
	cfg.Hardforks.Redelegations = append(cfg.Hardforks.Redelegations, chain_config.Redelegation{Validator: validator2_addr, Delegator: delegator3_addr, Amount: DefaultMinimumDeposit})
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	/*
		Simulate scenario when we have:

		  - total unique trxs count == 40
			- validator 1:
					- stake == 12.5% (from total stake)
					- he delegates to himself those 12.5%
					- added 8 unique trxs
					- 1 vote
			- validator 2:
					- stake == 62.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- added 32 unique trxs
					- 1 vote
			- delegator 3:
					- delegated 50% (from total stake) to validator 2
			- validator 4:
					- stake == 12.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- 1 vote
			- validator 5:
					- stake == 12.5% (from total stake)
					- he delegates to himself 12.5% (from total stake)
					- 1 vote

		After every participant claims his rewards:

			- block author (validator 1) - added 7 votes => bonus 1 vote
			- delegator 1(validator 1) gets 100 % from validator1_rewards
			- delegator 2(validator 2) gets 20 % from validator2_rewards
			- delegator 3 gets 80 % from validator2_rewards
			- delegator 4 gets 100% reward for 1 vote
	*/

	// Creates validators & delegators
	test.ExecuteAndCheck(validator1_owner, delegator1_stake, test.Pack("registerValidator", validator1_addr, validator1_proof, DefaultVrfKey, validator1_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake := delegator1_stake

	test.ExecuteAndCheck(validator2_owner, delegator2_stake, test.Pack("registerValidator", validator2_addr, validator2_proof, DefaultVrfKey, validator2_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator2_stake)

	test.ExecuteAndCheck(delegator3_addr, delegator3_stake, test.Pack("delegate", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator3_stake)

	test.ExecuteAndCheck(validator4_owner, delegator4_stake, test.Pack("registerValidator", validator4_addr, validator4_proof, DefaultVrfKey, validator4_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator4_stake)

	test.ExecuteAndCheck(validator5_owner, delegator5_stake, test.Pack("registerValidator", validator5_addr, validator5_proof, DefaultVrfKey, validator5_commission, "test", "test"), util.ErrorString(""), util.ErrorString(""))
	total_stake = bigutil.Add(total_stake, delegator5_stake)

	test.CheckContractBalance(total_stake)

	// Simulated rewards statistics
	tmp_rewards_stats := NewRewardsStats(&validator1_addr)

	validator1_stats := rewards_stats.ValidatorStats{}
	validator1_stats.DagBlocksCount = 8
	validator1_stats.VoteWeight = 1
	validator1_stats.FeesRewards = big.NewInt(int64(validator1_stats.DagBlocksCount))
	validator1_stats.FeesRewards.Mul(validator1_stats.FeesRewards, trxFee)
	tmp_rewards_stats.ValidatorsStats[validator1_addr] = validator1_stats

	validator2_stats := rewards_stats.ValidatorStats{}
	validator2_stats.DagBlocksCount = 32
	validator2_stats.VoteWeight = 5
	validator2_stats.FeesRewards = big.NewInt(int64(validator2_stats.DagBlocksCount))
	validator2_stats.FeesRewards.Mul(validator2_stats.FeesRewards, trxFee)
	tmp_rewards_stats.ValidatorsStats[validator2_addr] = validator2_stats

	validator4_stats := rewards_stats.ValidatorStats{}
	validator4_stats.VoteWeight = 1
	tmp_rewards_stats.ValidatorsStats[validator4_addr] = validator4_stats

	tmp_rewards_stats.TotalDagBlocksCount = validator1_stats.DagBlocksCount + validator2_stats.DagBlocksCount
	tmp_rewards_stats.TotalVotesWeight = 7
	tmp_rewards_stats.MaxVotesWeight = 8

	// Advance block
	reward := test.AdvanceBlock(&validator1_addr, &tmp_rewards_stats).ToBig()
	totalBalance := bigutil.Add(total_stake, reward)
	numberOfTrxs := new(big.Int)
	numberOfTrxs.SetUint64(uint64(tmp_rewards_stats.TotalDagBlocksCount))
	totalBalance.Add(totalBalance, bigutil.Mul(trxFee, numberOfTrxs))
	test.CheckContractBalance(totalBalance)

	// Expected block reward
	expected_block_reward := bigutil.Mul(total_stake, big.NewInt(int64(test.Chain_cfg.DPOS.YieldPercentage)))
	expected_block_reward = bigutil.Div(expected_block_reward, bigutil.Mul(big.NewInt(100), big.NewInt(int64(test.Chain_cfg.DPOS.BlocksPerYear))))

	// Splitting block rewards between votes and blocks
	expected_dag_reward := bigutil.Div(bigutil.Mul(expected_block_reward, big.NewInt(int64(test.Chain_cfg.DPOS.DagProposersReward))), big.NewInt(100))
	expected_vote_reward := bigutil.Sub(expected_block_reward, expected_dag_reward)

	// Vote bonus rewards - aka Author reward
	maxBlockAuthorReward := big.NewInt(int64(DefaultChainCfg.DPOS.MaxBlockAuthorReward))
	bonus_reward := bigutil.Div(bigutil.Mul(expected_block_reward, maxBlockAuthorReward), big.NewInt(100))
	expected_vote_reward = bigutil.Sub(expected_vote_reward, bonus_reward)

	// Vote bonus rewards - aka Author reward
	max_votes_weigh := dpos.Max(tmp_rewards_stats.MaxVotesWeight, tmp_rewards_stats.TotalVotesWeight)
	threshold := max_votes_weigh*2/3 + 1
	author_reward := bigutil.Div(bigutil.Mul(bonus_reward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight-threshold))), big.NewInt(int64(max_votes_weigh-threshold)))

	// Expected participants rewards
	// validator1_rewards = (validator1_trxs * blockReward) / total_trxs
	validator1_total_reward := bigutil.Div(bigutil.Mul(expected_dag_reward, big.NewInt(int64(validator1_stats.DagBlocksCount))), big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount)))
	// Add vote reward
	validatorVoteReward := bigutil.Mul(big.NewInt(int64(validator1_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	validator1_total_reward = bigutil.Add(validator1_total_reward, validatorVoteReward)
	// Commission reward
	expected_validator1_commission_reward := bigutil.Div(bigutil.Mul(validator1_total_reward, big.NewInt(int64(validator1_commission))), big.NewInt(10000))
	expected_validator1_delegators_reward := bigutil.Sub(validator1_total_reward, expected_validator1_commission_reward)

	// Fee rewards goes to commission pool
	expected_validator1_commission_reward = bigutil.Add(expected_validator1_commission_reward, bigutil.Mul(trxFee, big.NewInt(int64(validator1_stats.DagBlocksCount))))

	// Add author reward
	author_commission_reward := bigutil.Div(bigutil.Mul(author_reward, big.NewInt(int64(validator1_commission))), big.NewInt(10000))
	author_reward = bigutil.Sub(author_reward, author_commission_reward)
	expected_validator1_delegators_reward = bigutil.Add(expected_validator1_delegators_reward, author_reward)
	expected_validator1_commission_reward = bigutil.Add(expected_validator1_commission_reward, author_commission_reward)

	// validator2_rewards = (validator2_trxs * blockReward) / total_trxs
	validator2_total_reward := bigutil.Div(bigutil.Mul(expected_dag_reward, big.NewInt(int64(validator2_stats.DagBlocksCount))), big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount)))
	// Add vote reward
	validatorVoteReward = bigutil.Mul(big.NewInt(int64(validator2_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	validator2_total_reward = bigutil.Add(validator2_total_reward, validatorVoteReward)

	expected_validator2_commission_reward := bigutil.Div(bigutil.Mul(validator2_total_reward, big.NewInt(int64(validator2_commission))), big.NewInt(10000))
	expected_validator2_delegators_reward := bigutil.Sub(validator2_total_reward, expected_validator2_commission_reward)

	// Fee rewards goes to commission pool
	expected_validator2_commission_reward = bigutil.Add(expected_validator2_commission_reward, bigutil.Mul(trxFee, big.NewInt(int64(validator2_stats.DagBlocksCount))))

	// Add vote reward for validator 4
	validatorVoteReward = bigutil.Mul(big.NewInt(int64(validator4_stats.VoteWeight)), expected_vote_reward)
	validatorVoteReward = bigutil.Div(validatorVoteReward, big.NewInt(int64(tmp_rewards_stats.TotalVotesWeight)))
	expected_delegator4_reward := validatorVoteReward

	// delegator 1(validator 1) gets 100 % from validator1_rewards
	expected_delegator1_reward := expected_validator1_delegators_reward

	// delegator 2(validator 2) gets 20 % from validator2_rewards
	expected_delegator2_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(20)), big.NewInt(100))

	// delegator 3 gets 80 % from validator2_rewards
	expected_delegator3_reward := bigutil.Div(bigutil.Mul(expected_validator2_delegators_reward, big.NewInt(80)), big.NewInt(100))

	// expected_dag_rewardPlusFees := bigutil.Add(expected_dag_reward, bigutil.Mul(trxFee, big.NewInt(int64(tmp_rewards_stats.TotalDagBlocksCount))))
	// expectedDelegatorsRewards := bigutil.Add(expected_delegator1_reward, bigutil.Add(expected_delegator2_reward, expected_delegator3_reward))
	// // Last digit is removed due to rounding error that makes these values unequal
	// tc.Assert.Equal(bigutil.Div(expected_dag_rewardPlusFees, big.NewInt(1)0), bigutil.Div(expectedDelegatorsRewards, big.NewInt(1)0))

	// ErrNonExistentDelegation
	test.ExecuteAndCheck(validator1_owner, big.NewInt(0), test.Pack("claimRewards", validator2_addr), dpos.ErrNonExistentDelegation, util.ErrorString(""))

	// Check delgators rewards
	delegator1_old_balance := test.GetBalance(&delegator1_addr)
	delegator2_old_balance := test.GetBalance(&delegator2_addr)
	delegator3_old_balance := test.GetBalance(&delegator3_addr)
	delegator4_old_balance := test.GetBalance(&delegator4_addr)
	delegator4_old_balance.Sub(delegator4_old_balance, DefaultMinimumDeposit)

	// Check getter
	batch0_result := test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("getDelegations", delegator1_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
	batch0_parsed_result := new(GetDelegationsRet)
	test.Unpack(batch0_parsed_result, "getDelegations", batch0_result.CodeRetval)
	tc.Assert.Equal(1, len(batch0_parsed_result.Delegations))
	tc.Assert.Equal(true, batch0_parsed_result.End)

	validator_raw := test.ExecuteAndCheck(validator2_addr, big.NewInt(0), test.Pack("getValidator", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	validator := new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	old_stake := validator.ValidatorInfo.TotalStake

	{
		batch0_result := test.ExecuteAndCheck(delegator3_addr, big.NewInt(0), test.Pack("getDelegations", delegator3_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
		batch0_parsed_result := new(GetDelegationsRet)
		test.Unpack(batch0_parsed_result, "getDelegations", batch0_result.CodeRetval)
		tc.Assert.Equal(1, len(batch0_parsed_result.Delegations))
		tc.Assert.Equal(true, batch0_parsed_result.End)
		old_delegation := batch0_parsed_result.Delegations[0].Delegation.Stake

		//Redelegate issue
		test.ExecuteAndCheck(delegator3_addr, big.NewInt(0), test.Pack("reDelegate", validator2_addr, validator2_addr, DefaultMinimumDeposit), util.ErrorString(""), util.ErrorString(""))

		// HF happens
		test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("claimRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))

		batch0_result = test.ExecuteAndCheck(delegator3_addr, big.NewInt(0), test.Pack("getDelegations", delegator3_addr, uint32(0) /* batch */), util.ErrorString(""), util.ErrorString(""))
		batch0_parsed_result = new(GetDelegationsRet)
		test.Unpack(batch0_parsed_result, "getDelegations", batch0_result.CodeRetval)
		tc.Assert.Equal(1, len(batch0_parsed_result.Delegations))
		tc.Assert.Equal(true, batch0_parsed_result.End)
		new_delegation := batch0_parsed_result.Delegations[0].Delegation.Stake
		tc.Assert.Equal(new_delegation, old_delegation)
	}

	validator_raw = test.ExecuteAndCheck(validator2_addr, big.NewInt(0), test.Pack("getValidator", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	validator = new(GetValidatorRet)
	test.Unpack(validator, "getValidator", validator_raw.CodeRetval)
	new_stake := validator.ValidatorInfo.TotalStake
	tc.Assert.Equal(new_stake, old_stake)

	{
		test.ExecuteAndCheck(delegator3_addr, big.NewInt(0), test.Pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
		clam_res := test.ExecuteAndCheck(delegator4_addr, DefaultMinimumDeposit, test.Pack("delegate", validator4_addr), util.ErrorString(""), util.ErrorString(""))
		tc.Assert.Equal(len(clam_res.Logs), 2)
		tc.Assert.Equal(clam_res.Logs[0].Topics[0], RewardsClaimedEventHash)
		tc.Assert.Equal(clam_res.Logs[1].Topics[0], DelegatedEventHash)
	}

	test.ExecuteAndCheck(delegator2_addr, big.NewInt(0), test.Pack("claimRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))

	actual_delegator1_reward := bigutil.Sub(test.GetBalance(&delegator1_addr), delegator1_old_balance)
	actual_delegator2_reward := bigutil.Sub(test.GetBalance(&delegator2_addr), delegator2_old_balance)
	actual_delegator3_reward := bigutil.Sub(test.GetBalance(&delegator3_addr), delegator3_old_balance)
	actual_delegator4_reward := bigutil.Sub(test.GetBalance(&delegator4_addr), delegator4_old_balance)

	//Check claim vs getter result
	tc.Assert.Equal(batch0_parsed_result.Delegations[0].Delegation.Rewards, actual_delegator1_reward)

	tc.Assert.NotEqual(expected_delegator1_reward.Cmp(actual_delegator1_reward), 1)
	tc.Assert.NotEqual(expected_delegator2_reward.Cmp(actual_delegator2_reward), 1)
	tc.Assert.NotEqual(expected_delegator3_reward.Cmp(actual_delegator3_reward), 1)
	tc.Assert.NotEqual(expected_delegator4_reward.Cmp(actual_delegator4_reward), 1)

	// Check commission rewards
	validator1_old_balance := test.GetBalance(&validator1_owner)
	validator2_old_balance := test.GetBalance(&validator2_owner)
	validator4_old_balance := test.GetBalance(&validator4_owner)

	test.ExecuteAndCheck(delegator1_addr, big.NewInt(0), test.Pack("claimCommissionRewards", validator1_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(delegator2_addr, big.NewInt(0), test.Pack("claimCommissionRewards", validator2_addr), util.ErrorString(""), util.ErrorString(""))
	{
		claim_res := test.ExecuteAndCheck(delegator4_addr, big.NewInt(0), test.Pack("claimCommissionRewards", validator4_addr), util.ErrorString(""), util.ErrorString(""))
		tc.Assert.Equal(len(claim_res.Logs), 1)
		tc.Assert.Equal(claim_res.Logs[0].Topics[0], CommissionRewardsClaimedEventHash)
	}

	actual_validator1_commission_reward := bigutil.Sub(test.GetBalance(&validator1_owner), validator1_old_balance)
	actual_validator2_commission_reward := bigutil.Sub(test.GetBalance(&validator2_owner), validator2_old_balance)
	actual_validator4_commission_reward := bigutil.Sub(test.GetBalance(&validator4_owner), validator4_old_balance)

	tc.Assert.Equal(expected_validator1_commission_reward, actual_validator1_commission_reward)
	tc.Assert.Equal(expected_validator2_commission_reward, actual_validator2_commission_reward)
	tc.Assert.Equal(big.NewInt(0).Cmp(actual_validator4_commission_reward), 0)
	contractBalance := test.GetBalance(dpos.ContractAddress())
	if contractBalance.Cmp(total_stake) == -1 {
		t.Errorf("Balance left %d expected: %d", contractBalance, total_stake)
	}
}

func TestPhalaenopsisHF(t *testing.T) {
	cfg := CopyDefaultChainConfig()
	cfg.Hardforks.PhalaenopsisHfBlockNum = 3
	tc, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	testingAccount := addr(1)
	testingAccountBalance := test.GetBalance(&testingAccount)
	burnAmount := big.NewInt(1000000)
	test.ExecuteAndCheck(testingAccount, big.NewInt(1000000), dpos.TransferIntoDPoSContractMethod, util.ErrorString("no method with id: 0x44df8e70"), util.ErrorString(""))
	tc.Assert.Equal(testingAccountBalance, test.GetBalance(&testingAccount))

	test.AdvanceBlock(nil, nil)
	test.AdvanceBlock(nil, nil)

	dposBalanceBefore := test.GetBalance(dpos.ContractAddress())
	test.ExecuteAndCheck(testingAccount, big.NewInt(1000000), dpos.TransferIntoDPoSContractMethod, util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(testingAccountBalance.Sub(testingAccountBalance, burnAmount), test.GetBalance(&testingAccount))
	tc.Assert.Equal(dposBalanceBefore.Add(dposBalanceBefore, burnAmount), test.GetBalance(dpos.ContractAddress()))

	// totalBalance := bigutil.Add(total_stake, reward)

}

func TestNonPayableMethods(t *testing.T) {
	cfg := CopyDefaultChainConfig()
	cfg.Hardforks.CornusHf.BlockNum = 0
	_, test := test_utils.Init_test(dpos.ContractAddress(), dpos_sol.TaraxaDposClientMetaData, t, cfg)
	defer test.End()

	nonPayableMethods := []string{"undelegate", "undelegateV2", "confirmUndelegate", "confirmUndelegateV2", "cancelUndelegate", "cancelUndelegateV2", "reDelegate", "claimCommissionRewards", "setCommission", "setValidatorInfo", "isValidatorEligible", "getTotalEligibleVotesCount", "getValidatorEligibleVotesCount", "getValidator", "claimRewards", "claimAllRewards", "getValidators", "getValidatorsFor", "getTotalDelegation", "getDelegations", "getUndelegations", "getUndelegationsV2", "getUndelegationV2"}

	caller := addr(1)
	for _, method := range nonPayableMethods {
		test.ExecuteAndCheck(caller, big.NewInt(1), test.MethodId(method), dpos.ErrNonPayableMethod, util.ErrorString(""))
	}
}
