package slashing

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type API struct {
	config_by_block []DposConfigWithBlock
	config          chain_config.ChainConfig
}

type DposConfigWithBlock struct {
	DposConfig chain_config.DPOSConfig
	Blk_n      types.BlockNum
}

func (self *API) Init(cfg chain_config.ChainConfig) *API {
	self.config = cfg
	return self
}

func (self *API) GetConfigByBlockNum(blk_n uint64) chain_config.ChainConfig {
	for i, e := range self.config_by_block {
		// numeric_limits::max
		next_block_num := ^uint64(0)
		l_size := len(self.config_by_block)
		if i < l_size-1 {
			next_block_num = self.config_by_block[i+1].Blk_n
		}
		if (e.Blk_n <= blk_n) && (next_block_num > blk_n) {
			cfg := self.config
			cfg.DPOS = e.DposConfig
			return cfg
		}
	}
	return self.config
}

func (self *API) UpdateConfig(blk_n types.BlockNum, cfg chain_config.ChainConfig) {
	self.config_by_block = append(self.config_by_block, DposConfigWithBlock{cfg.DPOS, blk_n})
	self.config = cfg
}

func (self *API) NewContract(storage contract_storage.Storage, reader Reader, evm *vm.EVM) *Contract {
	return new(Contract).Init(self.config.Hardforks.MagnoliaHf, storage, reader, evm)
}

func (self *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret Reader) {
	cfg := self.GetConfigByBlockNum(blk_n)
	ret.Init(&cfg, blk_n, storage_factory)
	return
}
