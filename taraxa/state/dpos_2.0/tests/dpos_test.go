package test_integration

import (
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/core/vm"
	dpos_2 "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos_2.0/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
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

func TestUedelegate(t *testing.T) {
	_, test := init_config(t)
	defer test.end()

	test.ExecuteAndCheck(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"), util.ErrorString(""), util.ErrorString(""))
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(10)), util.ErrorString(""), util.ErrorString(""))
	//Check from same undelegate request
	test.ExecuteAndCheck(addr(1), 0, test.pack("undelegate", addr(1), new(big.Int).SetUint64(10)), dpos_2.ErrExistentUndelegation, util.ErrorString(""))
}
