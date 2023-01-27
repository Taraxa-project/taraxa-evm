package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

type Rewards struct {
	// Rewards accumulated
	RewardsPool *big.Int

	// Rewards accumulated
	CommissionRewardsPool *big.Int
}

func (self *Rewards) Empty() bool {
	return (self.RewardsPool.Cmp(big.NewInt(0)) == 0) && (self.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0)
}

type Validator struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission uint16

	// Block number related to commission
	LastCommissionChange types.BlockNum

	// Block number pointing to latest state
	LastUpdated types.BlockNum

	// Set of delegators
	Delegators map[common.Address]struct{}
}

func (self *Validator) AddDelegator(delegator *common.Address) {
	self.Delegators[*delegator] = struct{}{}
}

func (self *Validator) RemoveDelegator(delegator *common.Address) {
	delete(self.Delegators, *delegator)
}

type ValidatorInfo struct {
	// Validators description
	Description string

	// Validators website endpoint
	Endpoint string
}

// Validators type groups together all functionality related to creating/deleting/modifying/etc... validators
// as such info is stored under multiple independent storage keys, it is important that caller does not need to
// think about all implementation details, but just calls functions on Validators type
type Validators struct {
	storage         *StorageWrapper
	validators_list IterableMap

	validators_field        []byte
	validators_info_field   []byte
	validator_owner_field   []byte
	validator_vrf_key_field []byte
	validator_rewards_field []byte
}

var (
	main_index    = []byte{0}
	info_index    = []byte{1}
	owner_index   = []byte{2}
	vrf_index     = []byte{3}
	rewards_index = []byte{4}
	list_index    = []byte{5}
)

func (self *Validators) Init(stor *StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Validators storage fields keys - relative to the prefix
	self.validators_field = append(prefix, main_index...)
	self.validators_info_field = append(prefix, info_index...)
	self.validator_owner_field = append(prefix, owner_index...)
	self.validator_vrf_key_field = append(prefix, vrf_index...)
	self.validator_rewards_field = append(prefix, rewards_index...)
	validators_list_field := append(prefix, list_index...)

	self.validators_list.Init(self.storage, validators_list_field)
}

// Checks if correct account is trying to access validator object
func (self *Validators) CheckValidatorOwner(owner, validator *common.Address) bool {
	key := stor_k_1(self.validator_owner_field, validator[:])
	var saved_addr common.Address
	self.storage.Get(key, func(bytes []byte) {
		saved_addr = common.BytesToAddress(bytes)
	})
	return *owner == saved_addr
}

// Checks if correct account is trying to access validator object
func (self *Validators) GetValidatorOwner(validator *common.Address) (ret common.Address) {
	key := stor_k_1(self.validator_owner_field, validator[:])
	self.storage.Get(key, func(bytes []byte) {
		ret = common.BytesToAddress(bytes)
	})
	return
}

// Returns public vrf key for validator
func (self *Validators) GetVrfKey(validator *common.Address) (ret []byte) {
	key := stor_k_1(self.validator_vrf_key_field, validator[:])
	self.storage.Get(key, func(bytes []byte) {
		ret = bytes
	})
	return
}

// Checks is validator exists
func (self *Validators) ValidatorExists(validator_address *common.Address) bool {
	return self.validators_list.AccountExists(validator_address)
}

// Gets validator
func (self *Validators) GetValidator(validator_address *common.Address) (validator *Validator) {
	key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Get(key, func(bytes []byte) {
		validator = new(Validator)
		rlp.MustDecodeBytes(bytes, validator)
	})

	return
}

func (self *Validators) ModifyValidator(validator_address *common.Address, validator *Validator) {
	if validator == nil {
		panic("ModifyValidator: validator cannot be nil")
	}

	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("ModifyValidator: non existent validator")
	}

	key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(key, rlp.MustEncodeToBytes(validator))
}

func (self *Validators) CreateValidator(owner_address *common.Address, validator_address *common.Address, vrf_key []byte, block types.BlockNum, commission uint16, description string, endpoint string) *Validator {
	// Creates Validator object in storage
	validator := new(Validator)
	validator.Commission = commission
	validator.TotalStake = big.NewInt(0)
	validator.LastCommissionChange = block
	validator.LastUpdated = block

	validator_key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(validator_key, rlp.MustEncodeToBytes(validator))

	// Creates Validator_info object in storage
	validator_info := new(ValidatorInfo)
	validator_info.Description = description
	validator_info.Endpoint = endpoint

	validator_info_key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Put(validator_info_key, rlp.MustEncodeToBytes(validator_info))

	validator_owner_key := stor_k_1(self.validator_owner_field, validator_address[:])
	self.storage.Put(validator_owner_key, owner_address.Bytes())

	validator_vrf_key := stor_k_1(self.validator_vrf_key_field, validator_address[:])
	self.storage.Put(validator_vrf_key, vrf_key)

	rewards := new(Rewards)
	rewards.RewardsPool = big.NewInt(0)
	rewards.CommissionRewardsPool = big.NewInt(0)
	rewards_key := stor_k_1(self.validator_rewards_field, validator_address[:])
	self.storage.Put(rewards_key, rlp.MustEncodeToBytes(rewards))

	validator.Delegators = make(map[common.Address]struct{})
	// Adds validator into the list of all validators
	self.validators_list.CreateAccount(validator_address)
	return validator
}

func (self *Validators) DeleteValidator(validator_address *common.Address) {
	validator_key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(validator_key, nil)

	validator_info_key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Put(validator_info_key, nil)

	validator_owner_key := stor_k_1(self.validator_owner_field, validator_address[:])
	self.storage.Put(validator_owner_key, nil)

	validator_vrf_key := stor_k_1(self.validator_vrf_key_field, validator_address[:])
	self.storage.Put(validator_vrf_key, nil)

	rewards_key := stor_k_1(self.validator_rewards_field, validator_address[:])
	self.storage.Put(rewards_key, nil)

	// Removes validator into the list of all validators
	self.validators_list.RemoveAccount(validator_address)
}

func (self *Validators) GetValidatorInfo(validator_address *common.Address) (validator_info *ValidatorInfo) {
	key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Get(key, func(bytes []byte) {
		validator_info = new(ValidatorInfo)
		rlp.MustDecodeBytes(bytes, validator_info)
	})

	return
}

func (self *Validators) ModifyValidatorInfo(validator_address *common.Address, validator_info *ValidatorInfo) {
	if validator_info == nil {
		panic("ModifyValidatorInfo: validator_info cannot be nil")
	}

	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("ModifyValidatorInfo: non existent validator")
	}

	key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Put(key, rlp.MustEncodeToBytes(validator_info))
}

func (self *Validators) GetValidatorsAddresses(batch uint32, count uint32) ([]common.Address, bool) {
	return self.validators_list.GetAccounts(batch, count)
}

func (self *Validators) GetValidatorsCount() uint32 {
	return self.validators_list.GetCount()
}

func (self *Validators) GetValidatorRewards(validator_address *common.Address) (rewards *Rewards) {
	key := stor_k_1(self.validator_rewards_field, validator_address[:])
	self.storage.Get(key, func(bytes []byte) {
		rewards = new(Rewards)
		rlp.MustDecodeBytes(bytes, rewards)
	})

	return
}

func (self *Validators) ModifyValidatorRewards(validator_address *common.Address, rewards *Rewards) {
	if rewards == nil {
		panic("ModifyValidatorRewards: rewards cannot be nil")
	}

	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("ModifyValidatorRewards: non existent validator")
	}

	key := stor_k_1(self.validator_rewards_field, validator_address[:])
	self.storage.Put(key, rlp.MustEncodeToBytes(rewards))
}

func (self *Validators) AddValidatorRewards(validator_address *common.Address, commission_reward, reward *big.Int) {
	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("AddValidatorRewards: non existent validator")
	}
	rewards := self.GetValidatorRewards(validator_address)
	rewards.CommissionRewardsPool.Add(rewards.CommissionRewardsPool, commission_reward)
	rewards.RewardsPool.Add(rewards.RewardsPool, reward)

	self.ModifyValidatorRewards(validator_address, rewards)
}
