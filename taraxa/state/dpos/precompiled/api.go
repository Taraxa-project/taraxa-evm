package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

func ContractAddress() common.Address {
	return *contract_address
}

type API struct {
	cfg_by_block []ConfigWithBlock
	cfg          Config
}
type Config = struct {
	EligibilityBalanceThreshold *big.Int
	VoteEligibilityBalanceStep  *big.Int
	MaximumStake                *big.Int
	MinimumDeposit              *big.Int
	CommissionChangeDelta       uint16
	CommissionChangeFrequency   types.BlockNum
	DepositDelay                types.BlockNum
	WithdrawalDelay             types.BlockNum
	GenesisState                []GenesisStateEntry
}

type ConfigWithBlock struct {
	cfg   Config
	blk_n uint64
}
type GenesisStateEntry = struct {
	Benefactor common.Address
	Transfers  []GenesisTransfer
}
type GenesisTransfer = struct {
	Beneficiary common.Address
	Value       *big.Int
}

func (self *API) Init(cfg Config) *API {
	asserts.Holds(cfg.DepositDelay <= cfg.WithdrawalDelay)
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

func (self *API) UpdateConfig(blk_n uint64, cfg Config) {
	self.cfg_by_block = append(self.cfg_by_block, ConfigWithBlock{cfg, blk_n})
	self.cfg = cfg
}

func (self *API) NewContract(storage Storage, reader Reader) *Contract {
	return new(Contract).Init(self.cfg, storage, reader)
}

func (self *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) (ret Reader) {
	cfg := self.GetConfigByBlockNum(blk_n)
	ret.Init(&cfg, blk_n, storage_factory)
	return
}