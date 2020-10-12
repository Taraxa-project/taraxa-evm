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
	DepositDelay                types.BlockNum
	WithdrawalDelay             types.BlockNum
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

func (self *API) NewReader(blk_n types.BlockNum, backend_factory func(types.BlockNum) AccountStorageReader) Reader {
	if self.cfg.DepositDelay < blk_n {
		blk_n -= self.cfg.DepositDelay
	}
	return Reader{&self.cfg, backend_factory(blk_n)}
}

type Reader struct {
	cfg     *Config
	backend AccountStorageReader
}
type AccountStorageReader interface {
	GetAccountStorage(addr *common.Address, key *common.Hash, cb func([]byte))
}

func (self Reader) EligibleAddressCount() (ret uint64) {
	self.backend.GetAccountStorage(contract_address, stor_k(field_eligible_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) IsEligible(address *common.Address) bool {
	return self.GetStakingBalance(address).Cmp(self.cfg.EligibilityBalanceThreshold) >= 0
}

func (self Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = bigutil.Big0
	self.backend.GetAccountStorage(contract_address, stor_k(field_staking_balances, addr[:]), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func ContractAddress() common.Address {
	return *contract_address
}
