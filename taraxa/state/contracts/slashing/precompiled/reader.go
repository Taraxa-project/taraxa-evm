package slashing

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	slashing_sol "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/solidity"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type Reader struct {
	cfg     *chain_config.SlashingConfig
	storage *contract_storage.StorageReaderWrapper
}

func (self *Reader) Init(cfg *chain_config.SlashingConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) *Reader {
	self.cfg = cfg
	self.storage = new(contract_storage.StorageReaderWrapper).Init(slashing_contract_address, storage_factory(blk_n))
	return self
}

func (self *Reader) getJailInfo(addr *common.Address) (ret slashing_sol.SlashingInterfaceJailInfo) {
	var currrent_jail_block *types.BlockNum
	db_key := contract_storage.Stor_k_1(field_validators_jail_block, addr.Bytes())
	self.storage.Get(db_key, func(bytes []byte) {
		currrent_jail_block = new(types.BlockNum)
		rlp.MustDecodeBytes(bytes, currrent_jail_block)
	})

	ret.ProofsCount = 0
	if currrent_jail_block != nil {
		ret.JailBlock = big.NewInt(int64(*currrent_jail_block))
	} else {
		ret.JailBlock = big.NewInt(0)
	}

	return
}

func (self Reader) IsJailed(block types.BlockNum, addr *common.Address) bool {
	// TODO: check magnolia hardfork
	jailBlock := self.getJailInfo(addr).JailBlock
	if jailBlock.Uint64() >= block {
		return true
	}

	return false
}
