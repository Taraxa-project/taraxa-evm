package dpos

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type Config struct {
	EligibilityBalanceThreshold state_common.TaraxaBalance
	WithdrawalDelay             types.BlockNum
}
