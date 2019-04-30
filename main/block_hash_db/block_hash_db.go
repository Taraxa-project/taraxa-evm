package block_hash_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"strconv"
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
	value, err := this.db.Get(key(blockNumber))
	util.PanicOn(err)
	return common.HexToHash(string(value))
}

func (this *BlockHashDB) Put(blockNumber uint64, hash common.Hash) {
	this.db.Put(key(blockNumber), []byte(hash.Hex()))
}

func key(blockNumber uint64) []byte {
	return []byte(strconv.FormatUint(blockNumber, 10))
}
