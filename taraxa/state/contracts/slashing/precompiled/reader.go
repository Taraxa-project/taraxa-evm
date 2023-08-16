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

func (self *Reader) getJailInfo(addr *common.Address) (ret *slashing_sol.SlashingInterfaceJailInfo) {
	db_key := contract_storage.Stor_k_1(field_validators_jail_block, addr.Bytes())
	self.storage.Get(db_key, func(bytes []byte) {
		currrent_jail_block := new(types.BlockNum)
		rlp.MustDecodeBytes(bytes, currrent_jail_block)

		ret = new(slashing_sol.SlashingInterfaceJailInfo)
		ret.JailBlock = big.NewInt(int64(*currrent_jail_block))
		ret.ProofsCount = 0
	})

	return
}

func (self Reader) IsJailed(block types.BlockNum, addr *common.Address) bool {
	jail_info := self.getJailInfo(addr)
	if jail_info != nil && jail_info.JailBlock.Uint64() >= block {
		return true
	}

	return false
}
