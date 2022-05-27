package test_integration

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto/secp256k1"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

func TestProof(t *testing.T) {
	pubkey, seckey := generateKeyPair()
	addr := common.BytesToAddress(keccak256.Hash(pubkey[1:])[12:])
	proof, _ := secp256k1.Sign(addr.Hash().Bytes(), seckey)
	pubkey2, err := secp256k1.RecoverPubkey(addr.Hash().Bytes(), proof)
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
	_, test := init_test(t)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to register same validator twice
	test.ExecuteAndCheck(addr(2), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), dpos.ErrExistentValidator, util.ErrorString(""))
	// Try to register with not enough balance
	val_addr2, proof2 := generateAddrAndProof()
	test.ExecuteAndCheck(addr(2), 100000001, test.pack("registerValidator", val_addr2, proof2, uint16(10), "test", "test"), util.ErrorString(""), vm.ErrInsufficientBalanceForTransfer)
	// Try to register with wrong proof
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof2, uint16(10), "test", "test"), dpos.ErrWrongProof, util.ErrorString(""))
}

func TestDelegate(t *testing.T) {
	_, test := init_test(t)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to delegate to not existent validator
	test.ExecuteAndCheck(addr(1), 10, test.pack("delegate", addr(1)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(addr(1), 10, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
}

func TestRedelegate(t *testing.T) {
	_, test := init_test(t)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	val_addr1, proof1 := generateAddrAndProof()
	test.ExecuteAndCheck(addr(2), 10, test.pack("registerValidator", val_addr1, proof1, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(10)), util.ErrorString(""), util.ErrorString(""))
	//Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(10)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Validator to does not exist
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, addr(3), new(big.Int).SetUint64(10)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Validator from does not exist
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(3), val_addr, new(big.Int).SetUint64(10)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// Non existen delegation
	test.ExecuteAndCheck(addr(3), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(10)), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	// InsufficientDelegation
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(11)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(1)), util.ErrorString(""), util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(9)), util.ErrorString(""), util.ErrorString(""))
	// Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(10)), dpos.ErrNonExistentValidator, util.ErrorString(""))
}

func TestUndelegate(t *testing.T) {
	_, test := init_test(t)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", val_addr, new(big.Int).SetUint64(10)), util.ErrorString(""), util.ErrorString(""))
	// NonExistentValidator as it was deleted
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", val_addr, new(big.Int).SetUint64(10)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	//Check from same undelegate request
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", val_addr, new(big.Int).SetUint64(10)), dpos.ErrExistentUndelegation, util.ErrorString(""))
	// NonExistentValidator
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", addr(2), new(big.Int).SetUint64(10)), dpos.ErrNonExistentValidator, util.ErrorString(""))
	// NonExistentDelegation
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", val_addr, new(big.Int).SetUint64(10)), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	// ErrInsufficientDelegation
	test.ExecuteAndCheck(addr(2), 10, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", val_addr, new(big.Int).SetUint64(11)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
}

func TestRewards(t *testing.T) {
	tc, test := init_test(t)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(0), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	// ErrNonExistentDelegation
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimRewards", addr(2)), dpos.ErrNonExistentDelegation, util.ErrorString(""))
	old_balance := test.GetBalance(addr(1))
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimRewards", val_addr), util.ErrorString(""), util.ErrorString(""))
	new_balance := test.GetBalance(addr(1))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(30)), new_balance)
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimRewards", val_addr), util.ErrorString(""), util.ErrorString(""))
	new_balance = test.GetBalance(addr(1))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(30)), new_balance)
}

func TestCommissionRewards(t *testing.T) {
	tc, test := init_test(t)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(1000), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	// ErrNonExistentDelegation
	test.ExecuteAndCheck(addr(2), 0, test.pack("claimCommissionRewards", val_addr), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
	old_balance := test.GetBalance(addr(1))
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimCommissionRewards", val_addr), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(3)), test.GetBalance(addr(1)))
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimCommissionRewards", val_addr), util.ErrorString(""), util.ErrorString(""))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(3)), test.GetBalance(addr(1)))
}
func TestGenesis(t *testing.T) {
	genesis := DposGenesisState{
		addr(1): {
			addr(1): 1000,
			addr(2): 1000,
			addr(3): 1000,
		},
	}
	tc, test := init_test_genesis(t, genesis)
	defer test.end()

	tc.Assert.Equal(new(big.Int).SetUint64(100000000-3000), test.GetBalance(addr(1)))
	tc.Assert.Equal(uint64(3), test.GetDPOSReader().EligibleVoteCount())
	tc.Assert.Equal(new(big.Int).SetUint64(3000), test.GetDPOSReader().TotalAmountDelegated())
	tc.Assert.Equal(uint64(1), test.GetDPOSReader().GetEligibleVoteCount(addr_p(1)))
	tc.Assert.Equal(uint64(1), test.GetDPOSReader().GetEligibleVoteCount(addr_p(2)))
	tc.Assert.Equal(uint64(1), test.GetDPOSReader().GetEligibleVoteCount(addr_p(3)))
}

func TestSetCommissions(t *testing.T) {
	cfg := DposCfg{
		EligibilityBalanceThreshold: 1,
		CommissionChangeDelta:       5,
		CommissionChangeFrequency:   4,
	}
	_, test := init_test_config(t, cfg)
	defer test.end()

	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 0, test.pack("setCommission", val_addr, uint16(11)), dpos.ErrWrongOwnerAcc, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("setCommission", val_addr, uint16(11)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))
	//Advance 4 rounds
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.ExecuteAndCheck(addr(1), 0, test.pack("setCommission", val_addr, uint16(11)), util.ErrorString(""), util.ErrorString(""))
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{val_addr: new(big.Int).SetUint64(10)})
	//Advance 4 rounds
	test.ExecuteAndCheck(addr(1), 0, test.pack("setCommission", val_addr, uint16(20)), dpos.ErrForbiddenCommissionChange, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("setCommission", val_addr, uint16(16)), util.ErrorString(""), util.ErrorString(""))
}

func TestDelegateMinMax(t *testing.T) {
	cfg := DposCfg{
		EligibilityBalanceThreshold: 1,
		MinimumDeposit:              5,
		MaximumStake:                50,
	}
	_, test := init_test_config(t, cfg)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 1, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(addr(2), 1, test.pack("delegate", val_addr), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 40, test.pack("delegate", val_addr), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 10, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))
}

func TestUndelegateMin(t *testing.T) {
	cfg := DposCfg{
		EligibilityBalanceThreshold: 1,
		MinimumDeposit:              5,
	}
	_, test := init_test_config(t, cfg)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", val_addr,  new(big.Int).SetUint64(6)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 10, test.pack("delegate", val_addr), util.ErrorString(""), util.ErrorString(""))

	test.ExecuteAndCheck(addr(1), 1, test.pack("undelegate", val_addr,  new(big.Int).SetUint64(1)), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 1, test.pack("undelegate", val_addr,  new(big.Int).SetUint64(10)), util.ErrorString(""), util.ErrorString(""))
}

func TestRedelegateMinMax(t *testing.T) {
	cfg := DposCfg{
		EligibilityBalanceThreshold: 1,
		MinimumDeposit:              5,
		MaximumStake:                50,
	}
	_, test := init_test_config(t, cfg)
	defer test.end()
	val_addr, proof := generateAddrAndProof()
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", val_addr, proof, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	val_addr1, proof1 := generateAddrAndProof()
	test.ExecuteAndCheck(addr(2), 10, test.pack("registerValidator", val_addr1, proof1, uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(6)), dpos.ErrInsufficientDelegation, util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 35, test.pack("delegate", val_addr1), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(10)), dpos.ErrValidatorsMaxStakeExceeded, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", val_addr, val_addr1, new(big.Int).SetUint64(1)),  util.ErrorString(""), util.ErrorString(""))
}

// TODO undelegation test time wise
