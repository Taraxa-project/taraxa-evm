package test_integration

import (
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	dpos_2 "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos_2.0/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

func TestRegisterValidator(t *testing.T) {
	_, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to register same validator twice
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), dpos_2.ErrExistentValidator, util.ErrorString(""))
	// Try to register with not enough balance
	test.ExecuteAndCheck(addr(2), 100000001, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), vm.ErrInsufficientBalanceForTransfer)
}

func TestDelegate(t *testing.T) {
	_, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Try to delegate to not existent validator
	test.ExecuteAndCheck(addr(1), 10, test.pack("delegate", addr(2)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(addr(1), 10, test.pack("delegate", addr(1)), util.ErrorString(""), util.ErrorString(""))
}

func TestRedelegate(t *testing.T) {
	_, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(10)), util.ErrorString(""), util.ErrorString(""))
	//Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	// Validator to does not exist
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(3), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	// Validator from does not exist
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(3), addr(1), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	// Non existen delegation
	test.ExecuteAndCheck(addr(3), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentDelegation, util.ErrorString(""))
	// InsufficientDelegation
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(11)), dpos_2.ErrInsufficientDelegation, util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(1)), util.ErrorString(""), util.ErrorString(""))
	// OK
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(9)), util.ErrorString(""), util.ErrorString(""))
	// Validator 1 does not exist as we widthrawl all stake
	test.ExecuteAndCheck(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
}

func TestUndelegate(t *testing.T) {
	_, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(10)), util.ErrorString(""), util.ErrorString(""))
	// NonExistentValidator as it was deleted
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	//Check from same undelegate request
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(10)), dpos_2.ErrExistentUndelegation, util.ErrorString(""))
	// NonExistentValidator
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", addr(2), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	// NonExistentValidator as it was deleted
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", addr(2), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	// NonExistentDelegation
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(10)), dpos_2.ErrNonExistentDelegation, util.ErrorString(""))
	// ErrInsufficientDelegation
	test.ExecuteAndCheck(addr(2), 10, test.pack("delegate", addr(1)), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(2), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(11)), dpos_2.ErrInsufficientDelegation, util.ErrorString(""))
}

func TestRewards(t *testing.T) {
	tc, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(0), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.AddRewards(map[common.Address]*big.Int{addr(1): new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{addr(1): new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{addr(1): new(big.Int).SetUint64(10)})
	// ErrNonExistentDelegation
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimRewards", addr(2)), dpos_2.ErrNonExistentDelegation, util.ErrorString(""))
	old_balance := test.GetBalance(addr(1))
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimRewards", addr(1)), util.ErrorString(""), util.ErrorString(""))
	new_balance := test.GetBalance(addr(1))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(30)), new_balance)
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimRewards", addr(1)), util.ErrorString(""), util.ErrorString(""))
	new_balance = test.GetBalance(addr(1))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(30)), new_balance)
}

func TestCommissionRewards(t *testing.T) {
	tc, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(1000), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.AddRewards(map[common.Address]*big.Int{addr(1): new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{addr(1): new(big.Int).SetUint64(10)})
	test.AddRewards(map[common.Address]*big.Int{addr(1): new(big.Int).SetUint64(10)})
	// ErrNonExistentDelegation
	test.ExecuteAndCheck(addr(2), 0, test.pack("claimCommissionRewards"), dpos_2.ErrNonExistentValidator, util.ErrorString(""))
	old_balance := test.GetBalance(addr(1))
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimCommissionRewards"), util.ErrorString(""), util.ErrorString(""))
	new_balance := test.GetBalance(addr(1))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(3)), new_balance)
	test.ExecuteAndCheck(addr(1), 0, test.pack("claimCommissionRewards"), util.ErrorString(""), util.ErrorString(""))
	new_balance = test.GetBalance(addr(1))
	tc.Assert.Equal(bigutil.Add(old_balance, new(big.Int).SetUint64(3)), new_balance)
}

