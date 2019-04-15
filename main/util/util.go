package util

import "math/big"

func Sum(x, y *big.Int) *big.Int {
	return new(big.Int).Add(x, y)
}