package external_api

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/rawdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type ExternalApi struct {
	GetHeaderHashByBlockNumber func(u uint64) common.Hash
}

// TODO refactor
func New(blockchainDB rawdb.DatabaseReader) *ExternalApi {
	return &ExternalApi{
		GetHeaderHashByBlockNumber: func(u uint64) common.Hash {
			key := []byte(string(u))
			value, err := blockchainDB.Get(key)
			util.PanicOn(err)
			return common.BytesToHash(value)
		},
	}
}
