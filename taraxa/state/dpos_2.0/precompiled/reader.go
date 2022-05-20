package dpos_2

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type Reader struct {
	cfg     *Config
	storage *StorageReaderWrapper
}

func (self *Reader) Init(cfg *Config, blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) *Reader {
	self.cfg = cfg
	var blk_n_actual types.BlockNum
	if self.cfg.DepositDelay < blk_n {
		blk_n_actual = blk_n - self.cfg.DepositDelay
	} else {
		blk_n_actual = blk_n
	}
	self.storage = new(StorageReaderWrapper).Init(storage_factory(blk_n_actual))
	return self
}

func (self Reader) EligibleAddressCount() (ret uint64) {
	self.storage.Get(stor_k_1(field_eligible_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) EligibleVoteCount() (ret uint64) {
	self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) GetEligibleVoteCount(addr *common.Address) (ret uint64) {
	return vote_count(self.GetStakingBalance(addr), self.cfg.EligibilityBalanceThreshold, self.cfg.VoteEligibilityBalanceStep)
}

func (self Reader) TotalAmountDelegated() (ret *big.Int) {
	ret = bigutil.Big0
	self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func (self Reader) IsEligible(address *common.Address) bool {
	return self.cfg.EligibilityBalanceThreshold.Cmp(self.GetStakingBalance(address)) <= 0
}

func (self Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = bigutil.Big0
	self.storage.Get(stor_k_1(field_validators, addr[:]), func(bytes []byte) {
		validator := new(Validator)
		rlp.MustDecodeBytes(bytes, validator)
		ret = validator.TotalStake
	})
	return
}
