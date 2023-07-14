package slashing

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type API struct {
	cfg_by_block []ConfigWithBlock
	cfg          Config
}

type Config = struct {
	JailTime uint32 // [number of blocks]
}

type ConfigWithBlock struct {
	cfg   Config
	blk_n types.BlockNum
}

func (self *API) Init(cfg Config) *API {
	//asserts.Holds(cfg.JailTime > 0)
	self.cfg = cfg
	return self
}

func (self *API) GetConfigByBlockNum(blk_n uint64) Config {
	for i, e := range self.cfg_by_block {
		// numeric_limits::max
		next_block_num := ^uint64(0)
		l_size := len(self.cfg_by_block)
		if i < l_size-1 {
			next_block_num = self.cfg_by_block[i+1].blk_n
		}
		if (e.blk_n <= blk_n) && (next_block_num > blk_n) {
			return e.cfg
		}
	}
	return self.cfg
}

func (self *API) UpdateConfig(blk_n types.BlockNum, cfg Config) {
	self.cfg_by_block = append(self.cfg_by_block, ConfigWithBlock{cfg, blk_n})
	self.cfg = cfg
}

func (self *API) NewContract(storage contract_storage.Storage, reader Reader, evm *vm.EVM) *Contract {
	return new(Contract).Init(self.cfg, storage, reader, evm)
}

func (self *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) contract_storage.StorageReader) (ret Reader) {
	cfg := self.GetConfigByBlockNum(blk_n)
	ret.Init(&cfg, blk_n, storage_factory)
	return
}
