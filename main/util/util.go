package util

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"math/big"
)

func Sum(x, y *big.Int) *big.Int {
	if x == nil {
		x = common.Big0
	}
	if y == nil {
		y = common.Big0
	}
	return new(big.Int).Add(x, y)
}

func Noop(...interface{}) interface{} {
	return nil
}

func Chain(f, g func()) func() {
	return func() {
		f()
		g()
	}
}
