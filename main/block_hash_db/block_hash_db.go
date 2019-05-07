package block_hash_db

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type BlockHashDB struct {
	db ethdb.Database
}

func New(db ethdb.Database) *BlockHashDB {
	this := new(BlockHashDB)
	this.db = db
	return this
}

func (this *BlockHashDB) GetHeaderHashByBlockNumber(blockNumber uint64) common.Hash {
	blockNumberStr := fmt.Sprintf("%09d", blockNumber)
	value, err := this.db.Get([]byte(blockNumberStr))
	util.PanicIfPresent(err)
	header := new(struct {
		Hash *common.Hash `json:"hash"`
	})
	err = json.Unmarshal(value, header)
	util.PanicIfPresent(err)
	util.Assert(header.Hash != nil)
	return *header.Hash
}
