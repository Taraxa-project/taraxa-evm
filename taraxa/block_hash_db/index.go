package block_hash_db

import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/rawdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"strconv"
)

type BlockHashDB struct {
	db    rawdb.DatabaseReader
	cache taraxa.ConcurrentHashMap
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
	// TODO base 16?
	blockNumberStr := strconv.FormatUint(blockNumber, 10)
	value, err := this.db.Get([]byte(blockNumberStr))
	util.PanicIfNotNil(err)
	header := new(struct {
		Hash *common.Hash `json:"hash"`
	})
	err = json.Unmarshal(value, header)
	util.PanicIfNotNil(err)
	util.Assert(header.Hash != nil)
	hash := *header.Hash
	this.cache.Set(blockNumber, hash)
	return hash
}
