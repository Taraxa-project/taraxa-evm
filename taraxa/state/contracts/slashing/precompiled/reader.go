package slashing

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type Reader struct {
	dpos_config *chain_config.DPOSConfig
	storage     *contract_storage.StorageReaderWrapper
}

func (self *Reader) Init(cfg *chain_config.DPOSConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) *Reader {
	self.dpos_config = cfg

	blk_n_actual := uint64(0)
	if uint64(self.dpos_config.DelegationDelay) < blk_n {
		blk_n_actual = blk_n - uint64(self.dpos_config.DelegationDelay)
	}

	self.storage = new(contract_storage.StorageReaderWrapper).Init(slashing_contract_address, storage_factory(blk_n_actual))
	return self
}

func (self *Reader) getJailBlock(addr *common.Address) (jailed bool, block types.BlockNum) {
	block = 0
	jailed = false

	db_key := contract_storage.Stor_k_1(field_validators_jail_block, addr.Bytes())
	self.storage.Get(db_key, func(bytes []byte) {
		rlp.MustDecodeBytes(bytes, &block)
		jailed = true
	})

	return
}

func (self Reader) IsJailed(block types.BlockNum, addr *common.Address) bool {
	jailed, jail_block := self.getJailBlock(addr)
	if !jailed {
		return false
	}

	if jail_block < block {
		return false
	}

	return true
}
