package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type Reader struct {
	dpos_config  *chain_config.DposConfig
	storage      *StorageReaderWrapper
	blk_n_actual types.BlockNum
}

func (self *Reader) Init(dpos_config *chain_config.DposConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) *Reader {
	self.dpos_config = dpos_config
	self.blk_n_actual = uint64(0)
	if uint64(self.dpos_config.DelegationDelay) < blk_n {
		self.blk_n_actual = blk_n - uint64(self.dpos_config.DelegationDelay)
	}

	self.storage = new(StorageReaderWrapper).Init(storage_factory(self.blk_n_actual))
	return self
}

func (self Reader) EligibleVoteCount() (ret uint64) {
	self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) GetEligibleVoteCount(addr *common.Address) (ret uint64) {
	return voteCount(self.GetStakingBalance(addr), self.dpos_config.EligibilityBalanceThreshold, self.dpos_config.VoteEligibilityBalanceStep)
}

func (self Reader) TotalAmountDelegated() (ret *big.Int) {
	ret = big.NewInt(0)
	self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func (self Reader) IsEligible(address *common.Address) bool {
	return self.dpos_config.EligibilityBalanceThreshold.Cmp(self.GetStakingBalance(address)) <= 0
}

func (self Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = big.NewInt(0)
	self.storage.Get(stor_k_1(field_validators, validator_index, addr[:]), func(bytes []byte) {
		// Try to decode into post-hardfork extented Validator struct first
		validator := new(Validator)
		validator.ValidatorV1 = new(ValidatorV1)

		err := rlp.DecodeBytes(bytes, validator)
		if err != nil {
			// Try to decode into pre-hardfork ValidatorV1 struct first
			err = rlp.DecodeBytes(bytes, validator.ValidatorV1)
			validator.UndelegationsCount = 0
			if err != nil {
				// This should never happen
				panic("Unable to decode validator rlp")
			}
		}

		ret = validator.TotalStake
	})
	return
}

func (self Reader) GetVrfKey(addr *common.Address) (ret []byte) {
	self.storage.Get(stor_k_1(field_validators, validator_vrf_index, addr[:]), func(bytes []byte) {
		ret = bytes
	})
	return
}
