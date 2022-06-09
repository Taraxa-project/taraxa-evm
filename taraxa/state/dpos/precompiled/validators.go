package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

type Validator struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission uint16

	// Rewards accumulated
	RewardsPool *big.Int

	// Rewards accumulated
	CommissionRewardsPool *big.Int

	// Block number related to commission
	LastCommissionChange types.BlockNum

	// Block number poiting to latest state
	LastUpdated types.BlockNum
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

	validators_field      []byte
	validators_info_field []byte
	validator_owner_field []byte
}

func (self *Validators) Init(stor *StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Validators storage fields keys - relative to the prefix
	self.validators_field = append(prefix, []byte{0}...)
	self.validators_info_field = append(prefix, []byte{1}...)
	self.validator_owner_field = append(prefix, []byte{2}...)
	validators_list_field := append(prefix, []byte{3}...)

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
		panic("ModifyDelegation: validator cannot be nil")
	}

	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("ModifyValidator: non existent validator")
	}

	key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(key, rlp.MustEncodeToBytes(validator))
}

func (self *Validators) CreateValidator(owner_address *common.Address, validator_address *common.Address, block types.BlockNum, commission uint16, description string, endpoint string) *Validator {
	// Creates Validator object in storage
	validator := new(Validator)
	validator.CommissionRewardsPool = bigutil.Big0
	validator.RewardsPool = bigutil.Big0
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
