package external_api

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/rawdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"strconv"
)

type ExternalApi struct {
	GetHeaderHashByBlockNumber func(u uint64) common.Hash
}

// TODO refactor
func New(blockchainDB rawdb.DatabaseReader) *ExternalApi {
	return &ExternalApi{
		GetHeaderHashByBlockNumber: func(blockNumber uint64) common.Hash {
			blockNumberStr := strconv.FormatUint(blockNumber, 10)
			value, err := blockchainDB.Get([]byte(blockNumberStr))
			util.PanicOn(err)
			return common.HexToHash(string(value))
		},
	}
}
