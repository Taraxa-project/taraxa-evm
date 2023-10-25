package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/holiman/uint256"

	chain_config "github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	slashing "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/slashing/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"

	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type Reader struct {
	cfg             *chain_config.ChainConfig
	storage_factory func(types.BlockNum) storage.StorageReader
	delayed_storage *storage.StorageReaderWrapper
	current_storage *storage.StorageReaderWrapper
	slashing_reader *slashing.Reader
}

func (r *Reader) Init(cfg *chain_config.ChainConfig, blk_n types.BlockNum, storage_factory func(types.BlockNum) storage.StorageReader) *Reader {
	r.cfg = cfg
	r.storage_factory = storage_factory
	blk_n_actual := uint64(0)
	if uint64(r.cfg.DPOS.DelegationDelay) < blk_n {
		blk_n_actual = blk_n - uint64(r.cfg.DPOS.DelegationDelay)
	}

	r.delayed_storage = new(storage.StorageReaderWrapper).Init(dpos_contract_address, storage_factory(blk_n_actual))
	r.current_storage = new(storage.StorageReaderWrapper).Init(dpos_contract_address, storage_factory(blk_n))
	r.slashing_reader = new(slashing.Reader).Init(cfg, blk_n, r, storage_factory)
	return r
}

func (r Reader) TotalEligibleVoteCount() (ret uint64) {
	r.delayed_storage.Get(storage.Stor_k_1(field_eligible_vote_count), func(bytes []byte) {
		ret = bin.DEC_b_endian_compact_64(bytes)
	})
	for _, addr := range r.slashing_reader.GetJailedValidators() {
		ret -= r.GetEligibleVoteCount(&addr)
	}
	return
}

func (r Reader) GetEligibleVoteCount(addr *common.Address) (ret uint64) {
	return voteCount(r.GetStakingBalance(addr), r.cfg.DPOS.EligibilityBalanceThreshold, r.cfg.DPOS.VoteEligibilityBalanceStep)
}

func (r Reader) TotalAmountDelegated() (ret *big.Int) {
	return r.totalAmountDelegated(r.delayed_storage)
}

func (r Reader) TotalAmountDelegatedForBlock(blk_n types.BlockNum) *big.Int {
	stor := new(storage.StorageReaderWrapper).Init(dpos_contract_address, r.storage_factory(blk_n))
	return r.totalAmountDelegated(stor)
}

func (r Reader) totalAmountDelegated(stor *storage.StorageReaderWrapper) (ret *big.Int) {
	ret = big.NewInt(0)
	stor.Get(storage.Stor_k_1(field_amount_delegated), func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	return
}

func (r Reader) IsEligible(address *common.Address) bool {
	return r.cfg.DPOS.EligibilityBalanceThreshold.Cmp(r.GetStakingBalance(address)) <= 0
}

func (r Reader) GetStakingBalance(addr *common.Address) (ret *big.Int) {
	ret = big.NewInt(0)
	r.delayed_storage.Get(storage.Stor_k_1(field_validators, validator_index, addr[:]), func(bytes []byte) {
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

type ValidatorStake struct {
	Address    common.Address
	TotalStake *big.Int
}

func (r Reader) GetValidatorsTotalStakes() (ret []ValidatorStake) {
	reader := new(storage.AddressesIMapReader)
	reader.Init(r.storage, append(field_validators, validator_list_index...))

	validators, _ := reader.GetAccounts(0, reader.GetCount())

	for _, addr := range validators {
		ret = append(ret, ValidatorStake{Address: addr, TotalStake: r.GetStakingBalance(&addr)})
	}

	return
}

func (r Reader) GetVrfKey(addr *common.Address) (ret []byte) {
	r.delayed_storage.Get(storage.Stor_k_1(field_validators, validator_vrf_index, addr[:]), func(bytes []byte) {
		ret = bytes
	})
	return
}

// TODO: can we return float ???
func (r Reader) GetCurrentYield() uint64 {
	yield := uint256.NewInt(0)
	r.current_storage.Get(contract_storage.Stor_k_1(field_current_yield), func(bytes []byte) {
		yield = new(uint256.Int).SetBytes(bytes)
	})

	// Float percentage yield would be
	//yield_percentage := float64(yield.ToBig().Int64()) / float64(YieldDecimalPrecision.ToBig().Int64())

	return yield.Uint64()
}
