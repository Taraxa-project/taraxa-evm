package slashing

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type Reader struct {
	cfg     *Config
	storage *contract_storage.StorageReaderWrapper
}

func (self *Reader) Init(cfg *Config, blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) *Reader {
	self.cfg = cfg
	self.storage = new(contract_storage.StorageReaderWrapper).Init(slashing_contract_address, storage_factory(blk_n))
	return self
}

func (self Reader) IsJailed() bool {
	return false
	// self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
	// 	jailed_until := bin.DEC_b_endian_compact_64(bytes)
	// })

	// return jailed_until >= actual_block
}
