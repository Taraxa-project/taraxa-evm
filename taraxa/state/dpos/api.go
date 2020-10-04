package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

type API struct {
	cfg Config
}
type Config = struct {
	EligibilityBalanceThreshold *big.Int
	WithdrawalDelay             types.BlockNum
	DepositDelay                types.BlockNum
	GenesisState                DelegatedBalanceMap
}

func (self *API) Init(cfg Config) *API {
	assert.Holds(cfg.DepositDelay <= cfg.WithdrawalDelay)
	self.cfg = cfg
	return self
}

func (self *API) NewContract(storage Storage) *Contract {
	return new(Contract).init(self.cfg, storage)
}

type AccountStorageByBlock = func(types.BlockNum, *common.Address, *common.Hash, func([]byte))

func (self *API) EligibleAddressCount(blk_n types.BlockNum, get_storage AccountStorageByBlock) (ret uint64) {
	get_storage(self.true_blk_n(blk_n),
		ContractAddress,
		stor_k(field_eligible_count),
		func(bytes []byte) {
			ret = bin.DEC_b_endian_compact_64(bytes)
		})
	return
}

func (self *API) IsEligible(blk_n types.BlockNum, address *common.Address, get_storage AccountStorageByBlock) bool {
	return self.GetStakingBalance(blk_n, address, get_storage).Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
}

func (self *API) GetStakingBalance(blk_n types.BlockNum, addr *common.Address, get_storage AccountStorageByBlock) (ret *big.Int) {
	ret = bigutil.Big0
	get_storage(
		self.true_blk_n(blk_n),
		ContractAddress,
		stor_k(field_staking_balances, addr[:]),
		func(bytes []byte) {
			ret = bigutil.FromBytes(bytes)
		})
	return
}

func (self *API) true_blk_n(client_blk_n types.BlockNum) (ret uint64) {
	if self.cfg.DepositDelay < client_blk_n {
		ret = client_blk_n - self.cfg.DepositDelay
	}
	return
}
