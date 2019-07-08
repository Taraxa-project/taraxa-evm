package block_hash_db

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/rawdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/cornelk/hashmap"
)

type BlockHashDB struct {
	db    rawdb.DatabaseReader
	cache hashmap.HashMap
}

func New(db rawdb.DatabaseReader) *BlockHashDB {
	this := new(BlockHashDB)
	this.db = db
	return this
}

func (this *BlockHashDB) GetHeaderHashByBlockNumber(blockNumber uint64) common.Hash {
	if val, cached := this.cache.Get(blockNumber); cached {
		return val.(common.Hash)
	}
	blockNumberStr := fmt.Sprintf("%09d", blockNumber)
	value, err := this.db.Get([]byte(blockNumberStr))
	util.PanicIfPresent(err)
	header := new(struct {
		Hash *common.Hash `json:"hash"`
	})
	err = json.Unmarshal(value, header)
	util.PanicIfPresent(err)
	util.Assert(header.Hash != nil)
	hash := *header.Hash
	this.cache.Set(blockNumber, hash)
	return hash
}
