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
	cfg Config
}
type Config = struct {
	EligibilityBalanceThreshold *big.Int
	DepositDelay                types.BlockNum
	WithdrawalDelay             types.BlockNum
	GenesisState                DelegatedBalanceMap
}

func (self *API) Init(cfg Config) *API {
	asserts.Holds(cfg.DepositDelay <= cfg.WithdrawalDelay)
	self.cfg = cfg
	return self
}

func (self *API) NewContract(storage Storage) *Contract {
	return new(Contract).init(self.cfg, storage)
}

func (self *API) NewReader(blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) (ret Reader) {
	ret.Init(&self.cfg, blk_n, storage_factory)
	return
}
