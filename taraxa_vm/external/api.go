package external

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"math/big"
)

var (
	GetHeaderHashByBlockNumber func(u uint64) common.Hash = func(u uint64) common.Hash {
		return common.BigToHash(big.NewInt(int64(u)))
	}
)
