package slashing

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type Reader struct {
	cfg     *chain_config.ChainConfig
	storage *contract_storage.StorageReaderWrapper
}

func (r *Reader) Init(cfg *chain_config.ChainConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) *Reader {
	r.cfg = cfg

	blk_n_actual := uint64(0)
	if uint64(r.cfg.DPOS.DelegationDelay) < blk_n {
		blk_n_actual = blk_n - uint64(r.cfg.DPOS.DelegationDelay)
	}

	r.storage = new(contract_storage.StorageReaderWrapper).Init(slashing_contract_address, storage_factory(blk_n_actual))
	return r
}

func (r *Reader) getJailBlock(addr *common.Address) (jailed bool, block types.BlockNum) {
	db_key := contract_storage.Stor_k_1(field_validators_jail_block, addr.Bytes())
	r.storage.Get(db_key, func(bytes []byte) {
		rlp.MustDecodeBytes(bytes, &block)
		jailed = true
	})

	return
}

func (r Reader) IsJailed(block types.BlockNum, addr *common.Address) bool {
	if !r.cfg.Hardforks.IsMagnoliaHardfork(block) {
		return false
	}

	jailed, jail_block := r.getJailBlock(addr)
	if !jailed {
		return false
	}

	if jail_block < block {
		return false
	}

	return true
}

func (r Reader) GetJailedValidators() (jailed_validators []common.Address) {
	jailed_validators_key := common.BytesToHash(field_jailed_validators)
	r.storage.Get(&jailed_validators_key, func(bytes []byte) {
		if len(bytes) != 0 {
			rlp.MustDecodeBytes(bytes, &jailed_validators)
		}
	})

	return
}
