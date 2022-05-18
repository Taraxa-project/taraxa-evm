package test_integration

import (
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)


func TestRegisterValidator(t *testing.T) {
	tc, test := init_config(t)
	defer test.end()

	r := test.execute(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)

	r = test.execute(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString("Validator already exist"), r.ExecutionErr)

	r = test.execute(addr(2), 100000001, test.pack("registerValidator", uint16(10), "test", "test"))
	tc.Assert.Equal(util.ErrorString("insufficient balance for transfer"), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)
}


func TestDelegate(t *testing.T) {
	tc, test := init_config(t)
	defer test.end()

	r := test.execute(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)

	r = test.execute(addr(1), 10, test.pack("delegate", addr(2)))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString("Validator does not exist"), r.ExecutionErr)

	r = test.execute(addr(1), 10, test.pack("delegate", addr(1)))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)
}

func TestRedelegate(t *testing.T) {
	tc, test := init_config(t)
	defer test.end()

	r := test.execute(addr(1), 10, test.pack("registerValidator", uint16(10), "test", "test"))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)

	r = test.execute(addr(2), 10, test.pack("registerValidator", uint16(10), "test", "test"))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)

	r = test.execute(addr(1), 0, test.pack("reDelegate", addr(1), addr(2), new(big.Int).SetUint64(10)))
	tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)

	// r = test.execute(addr(1), 0, test.pack("reDelegate", addr(1), addr(3), new(big.Int).SetUint64(10)))
	// tc.Assert.Equal(util.ErrorString(""), r.ConsensusErr)
	// tc.Assert.Equal(util.ErrorString(""), r.ExecutionErr)

}
