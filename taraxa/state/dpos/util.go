package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
)

func udiv64(x, y *big.Int) uint64 {
	var tmp big.Int
	tmp.Div(x, y)
	asserts.Holds(tmp.IsUint64())
	return tmp.Uint64()
}
