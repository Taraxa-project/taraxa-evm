package data_types

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos_2.0/precompiled/iterable"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

// Validator basic info
type ValidatorBasicInfo struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission *big.Int

	// Rewards accumulated from delegators rewards based on commission
	CommissionRewards *big.Int

	// // Short description
	// // TODO: optional - maybe we dont want this ?
	// Description string

	// // Validator's website url
	// // TODO: optional - maybe we dont want this ?
	// Endpoint string
}

func (vb ValidatorBasicInfo) Serialize(pos *common.Hash, out iterable.StorageData) {
	out[pos] = vb.TotalStake.Bytes()
	pos.Inc()
	out[pos] = vb.Commission.Bytes()
	pos.Inc()
	out[pos] = vb.CommissionRewards.Bytes()
	pos.Inc()
}

func (vb ValidatorBasicInfo) Load(pos *common.Hash, get func(*common.Hash, func([]byte))) {
	get(pos, func(bytes []byte) {
		vb.TotalStake.FillBytes(bytes)
	})
	pos.Inc()
	get(pos, func(bytes []byte) {
		vb.Commission.FillBytes(bytes)
	})
	pos.Inc()
	get(pos, func(bytes []byte) {
		vb.CommissionRewards.FillBytes(bytes)
	})
	pos.Inc()
}

// Validator info
type ValidatorInfo struct {
	// Validtor basic info
	BasicInfo ValidatorBasicInfo

	// List of validator's delegators
	Delegators iterable.AddressMap[*DelegatorInfo]
}

func (vi ValidatorInfo) Load(pos *common.Hash, get func(*common.Hash, func(bytes []byte))) {
}

func (vi ValidatorInfo) Serialize(pos *common.Hash, out iterable.StorageData) {
	vi.BasicInfo.Serialize(keccak256.Hash(pos.Bytes()), out)

	// SerializeAddressMap(pos, vi.Delegators, out)
}

type DelegatorInfo struct {
	// Num of delegated tokens == delegator's stake
	Stake *big.Int

	// UnlockedStake == unlocked(undelegated) tokens that can be withdrawn now
	// TODO: in case we will send unlocked tokens to the delegator's balance automatically, we dont need this field
	UnlockedStake *big.Int

	// Accumulated rewards
	Rewards *big.Int

	// Undelegate request
	// TODO: rethink implementation of undelegations
	UndelegateRequests []*UndelegateRequest
}

func (di DelegatorInfo) Load(pos *common.Hash, get func(*common.Hash, func(bytes []byte))) {
}
func (di DelegatorInfo) Serialize(pos *common.Hash, out iterable.StorageData) {
	out[pos] = di.Stake.Bytes()
	pos.Inc()
	out[pos] = di.UnlockedStake.Bytes()
	pos.Inc()
	out[pos] = di.Rewards.Bytes()
	pos.Inc()

	iterable.SerializeArray(pos, di.UndelegateRequests, out)
}

type UndelegateRequest struct {
	// Num of tokens that delegator wants to undelegate
	Amount *big.Int

	// Block number when this unstake request can be confirmed(act block num + locking period)
	EligibleBlockNum *big.Int
}

func (ur *UndelegateRequest) Load(pos *common.Hash, get func(*common.Hash, func(bytes []byte))) {
}
func (ur *UndelegateRequest) Serialize(pos *common.Hash, out iterable.StorageData) {
	out[pos] = ur.Amount.Bytes()
	pos.Inc()
	out[pos] = ur.EligibleBlockNum.Bytes()
	pos.Inc()
}

type SerializableAddress struct {
	common.Address
}

func (sa *SerializableAddress) Load(pos *common.Hash, get func(*common.Hash, func(bytes []byte))) {
	get(pos, func(bytes []byte) {
		sa.Address = common.BytesToAddress(bytes)
	})
}
func (sa *SerializableAddress) Serialize(pos *common.Hash, out iterable.StorageData) {
	out[pos] = sa.Address.Bytes()
}

// Delegator's validators info
type DelegatorValidators struct {
	// List of validators addresses that delegator delegated to
	// Note: info about delegator's stake/reward, etc... is saved in ValidatorDelegators struct
	Validators []*SerializableAddress
}

func (dv DelegatorValidators) Load(pos *common.Hash, get func(*common.Hash, func(bytes []byte))) {
}
func (dv DelegatorValidators) Serialize(pos *common.Hash, out iterable.StorageData) {
	iterable.SerializeArray(pos, dv.Validators, out)
}
