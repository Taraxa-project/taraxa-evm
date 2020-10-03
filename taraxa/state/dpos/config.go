package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/core/types"
)

type Config struct {
	EligibilityBalanceThreshold *big.Int
	WithdrawalDelay             types.BlockNum
	DepositDelay                types.BlockNum
}
