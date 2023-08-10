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
	cfg     *chain_config.ChainConfig
	storage *StorageReaderWrapper
}

func (self *Reader) Init(cfg *chain_config.ChainConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) StorageReader) *Reader {
	self.cfg = cfg
	blk_n_actual := uint64(0)
	if uint64(self.cfg.DPOS.DelegationDelay) < blk_n {
		blk_n_actual = blk_n - uint64(self.cfg.DPOS.DelegationDelay)
	}

	self.storage = new(StorageReaderWrapper).Init(storage_factory(blk_n_actual))
	return self
}

func (self Reader) EligibleVoteCount() (ret uint64) {
	self.storage.Get(stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	return
}

func (self Reader) GetEligibleVoteCount(addr *common.Address) (ret uint64) {
	return voteCount(self.GetStakingBalance(addr), self.cfg.DPOS.EligibilityBalanceThreshold, self.cfg.DPOS.VoteEligibilityBalanceStep)
}

func (self Reader) TotalAmountDelegated() (ret *big.Int) {
	ret = big.NewInt(0)
	self.storage.Get(stor_k_1(field_amount_delegated), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func (self Reader) IsEligible(address *common.Address) bool {
	return self.cfg.DPOS.EligibilityBalanceThreshold.Cmp(self.GetStakingBalance(address)) <= 0
}

func (self Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = big.NewInt(0)
	self.storage.Get(stor_k_1(field_validators, validator_index, addr[:]), func(bytes []byte) {
		validator := new(Validator)
		rlp.MustDecodeBytes(bytes, validator)
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
