package data_types

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type StorageData = map[*common.Hash][]byte

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

func (vb ValidatorBasicInfo) Serialize(pos *common.Hash, out StorageData) {
	out[pos] = vb.TotalStake.Bytes()
	pos.Inc()
	out[pos] = vb.Commission.Bytes()
	pos.Inc()
	out[pos] = vb.CommissionRewards.Bytes()
}

// Validator info
type ValidatorInfo struct {
	// Validtor basic info
	BasicInfo ValidatorBasicInfo

	// List of validator's delegators
	Delegators map[common.Address]*DelegatorInfo
}

func (vi ValidatorInfo) Serialize(pos *common.Hash, out StorageData) {
	vi.BasicInfo.Serialize(keccak256.Hash(pos.Bytes()), out)
	pos = pos.Add(big.NewInt(1))
	for i, v := range vi.Delegators {
		key := keccak256.Hash(i.Hash().Bytes(), pos.Bytes())
		v.Serialize(key, out)
	}
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

func (di DelegatorInfo) Serialize(pos *common.Hash, out StorageData) {
	out[pos] = di.Stake.Bytes()
	pos.Inc()
	out[pos] = di.UnlockedStake.Bytes()
	pos.Inc()
	out[pos] = di.Rewards.Bytes()
	pos.Inc()

	// write size
	out[keccak256.Hash(big.NewInt(0).Bytes(), pos.Bytes())] = big.NewInt(int64(len(di.UndelegateRequests))).Bytes()
	//write elements
	for i, v := range di.UndelegateRequests {
		key := keccak256.Hash(big.NewInt(int64(i+1)).Bytes(), pos.Bytes())
		v.Serialize(key, out)
	}
}

type UndelegateRequest struct {
	// Num of tokens that delegator wants to undelegate
	Amount *big.Int

	// Block number when this unstake request can be confirmed(act block num + locking period)
	EligibleBlockNum *big.Int
}

func (ur UndelegateRequest) Serialize(pos *common.Hash, out StorageData) {
	out[pos] = ur.Amount.Bytes()
	pos.Inc()
	out[pos] = ur.EligibleBlockNum.Bytes()
}

// Delegator's validators info
type DelegatorValidators struct {
	// List of validators addresses that delegator delegated to
	// Note: info about delegator's stake/reward, etc... is saved in ValidatorDelegators struct
	Validators []common.Address
}

func (vi DelegatorValidators) Serialize(pos *common.Hash, out StorageData) {
}
