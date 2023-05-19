package dpos

import (
	"fmt"
	"math/big"

	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

// Pre-hardfork validator struct without UndelegationsCount member
type ValidatorV1 struct {
	// TotalStake == sum of all delegated tokens to the validator
	TotalStake *big.Int

	// Commission
	Commission uint16

	// Block number related to commission
	LastCommissionChange types.BlockNum

	// Block number pointing to latest state
	LastUpdated types.BlockNum
}

type Validator struct {
	*ValidatorV1

	// Number of ongoing/unclaimed undelegations from the validator
	UndelegationsCount uint16
}

type ValidatorInfo struct {
	// Validators description
	Description string

	// Validators website endpoint
	Endpoint string
}

type ValidatorRewards struct {
	// Rewards accumulated
	RewardsPool *big.Int

	// Rewards accumulated
	CommissionRewardsPool *big.Int
}

func (self *ValidatorRewards) Empty() bool {
	return (self.RewardsPool.Cmp(big.NewInt(0)) == 0) && (self.CommissionRewardsPool.Cmp(big.NewInt(0)) == 0)
}

// Validators type groups together all functionality related to creating/deleting/modifying/etc... validators
// as such info is stored under multiple independent storage keys, it is important that caller does not need to
// think about all implementation details, but just calls functions on Validators type
type Validators struct {
	storage         *contract_storage.StorageWrapper
	validators_list contract_storage.AddressesIMap

	validator_field         []byte
	validator_info_field    []byte
	validator_rewards_field []byte
	validator_owner_field   []byte
	validator_vrf_key_field []byte
}

var (
	validator_index         = []byte{0}
	validator_info_index    = []byte{1}
	validator_rewards_index = []byte{2}
	validator_owner_index   = []byte{3}
	validator_vrf_index     = []byte{4}
	validator_list_index    = []byte{5}
)

func (self *Validators) Init(stor *contract_storage.StorageWrapper, prefix []byte) *Validators {
	self.storage = stor

	// Init Validators storage fields keys - relative to the prefix
	self.validator_field = append(prefix, validator_index...)
	self.validator_info_field = append(prefix, validator_info_index...)
	self.validator_rewards_field = append(prefix, validator_rewards_index...)
	self.validator_owner_field = append(prefix, validator_owner_index...)
	self.validator_vrf_key_field = append(prefix, validator_vrf_index...)

	self.validators_list.Init(self.storage, append(prefix, validator_list_index...))

	return self
}

// Checks if correct account is trying to access validator object
func (self *Validators) CheckValidatorOwner(owner, validator *common.Address) bool {
	saved_addr := self.GetValidatorOwner(validator)
	return *owner == saved_addr
}

// Checks if correct account is trying to access validator object
func (self *Validators) GetValidatorOwner(validator *common.Address) (ret common.Address) {
	key := contract_storage.Stor_k_1(self.validator_owner_field, validator[:])
	self.storage.Get(key, func(bytes []byte) {
		ret = common.BytesToAddress(bytes)
	})
	return
}

// Returns public vrf key for validator
func (self *Validators) GetVrfKey(validator *common.Address) (ret []byte) {
	key := contract_storage.Stor_k_1(self.validator_vrf_key_field, validator[:])
	self.storage.Get(key, func(bytes []byte) {
		ret = bytes
	})
	return
}

// Checks is validator exists
func (self *Validators) ValidatorExists(validator_address *common.Address) bool {
	return self.validators_list.AccountExists(validator_address)
}

func (self *Validators) GetValidatorsAddresses(batch uint32, count uint32) ([]common.Address, bool) {
	return self.validators_list.GetAccounts(batch, count)
}

func (self *Validators) GetValidatorsCount() uint32 {
	return self.validators_list.GetCount()
}

func (self *Validators) CreateValidator(extended_validator bool, owner_address *common.Address, validator_address *common.Address, vrf_key []byte, block types.BlockNum, commission uint16, description string, endpoint string) (validator *Validator) {
	// Creates Validator object in storage
	validator = new(Validator)
	validator.ValidatorV1 = new(ValidatorV1)
	validator.Commission = commission
	validator.TotalStake = big.NewInt(0)
	validator.LastCommissionChange = block
	validator.LastUpdated = block
	validator.UndelegationsCount = 0

	if extended_validator {
		Save(self, validator_address, validator)
	} else {
		Save(self, validator_address, validator.ValidatorV1)
	}

	// Creates ValidatorInfo object in storage
	validator_info := new(ValidatorInfo)
	validator_info.Description = description
	validator_info.Endpoint = endpoint
	Save(self, validator_address, validator_info)

	// Creates ValidatorRewards object in storage
	rewards := new(ValidatorRewards)
	rewards.RewardsPool = big.NewInt(0)
	rewards.CommissionRewardsPool = big.NewInt(0)
	Save(self, validator_address, rewards)

	validator_owner_key := contract_storage.Stor_k_1(self.validator_owner_field, validator_address[:])
	self.storage.Put(validator_owner_key, owner_address.Bytes())

	validator_vrf_key := contract_storage.Stor_k_1(self.validator_vrf_key_field, validator_address[:])
	self.storage.Put(validator_vrf_key, vrf_key)

	// Adds validator into the list of all validators
	self.validators_list.CreateAccount(validator_address)

	return
}

func (self *Validators) DeleteValidator(validator_address *common.Address) {
	validator_key := contract_storage.Stor_k_1(self.validator_field, validator_address[:])
	self.storage.Put(validator_key, nil)

	validator_info_key := contract_storage.Stor_k_1(self.validator_info_field, validator_address[:])
	self.storage.Put(validator_info_key, nil)

	validator_owner_key := contract_storage.Stor_k_1(self.validator_owner_field, validator_address[:])
	self.storage.Put(validator_owner_key, nil)

	validator_vrf_key := contract_storage.Stor_k_1(self.validator_vrf_key_field, validator_address[:])
	self.storage.Put(validator_vrf_key, nil)

	rewards_key := contract_storage.Stor_k_1(self.validator_rewards_field, validator_address[:])
	self.storage.Put(rewards_key, nil)

	// Removes validator from the list of all validators
	self.validators_list.RemoveAccount(validator_address)
}

func (self *Validators) GetValidator(validator_address *common.Address) (validator *Validator) {
	key := stor_k_1(self.validator_field, validator_address[:])
	self.storage.Get(key, func(bytes []byte) {
		// Try to decode into post-hardfork extented Validator struct first
		validator = new(Validator)
		validator.ValidatorV1 = new(ValidatorV1)

		err := rlp.DecodeBytes(bytes, validator)
		if err != nil {
			// Try to decode into pre-hardfork ValidatorV1 struct
			err = rlp.DecodeBytes(bytes, validator.ValidatorV1)
			validator.UndelegationsCount = 0
			if err != nil {
				// This should never happen
				panic("Unable to decode validator rlp")
			}
		}
	})

	return
}

func (self *Validators) ModifyValidator(extended_validator bool, validator_address *common.Address, validator *Validator) {
	if extended_validator {
		Modify(self, validator_address, validator)
	} else {
		Modify(self, validator_address, validator.ValidatorV1)
	}
}

func (self *Validators) GetValidatorInfo(validator_address *common.Address) (validator_info *ValidatorInfo) {
	return Get[ValidatorInfo](self, validator_address)
}

func (self *Validators) ModifyValidatorInfo(validator_address *common.Address, validator_info *ValidatorInfo) {
	Modify(self, validator_address, validator_info)
}

func (self *Validators) GetValidatorRewards(validator_address *common.Address) (rewards *ValidatorRewards) {
	return Get[ValidatorRewards](self, validator_address)
}

func (self *Validators) ModifyValidatorRewards(validator_address *common.Address, rewards *ValidatorRewards) {
	Modify(self, validator_address, rewards)
}

func (self *Validators) AddValidatorRewards(validator_address *common.Address, commission_reward, reward *big.Int) {
	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("AddValidatorRewards: non existent validator")
	}
	rewards := Get[ValidatorRewards](self, validator_address)
	rewards.CommissionRewardsPool.Add(rewards.CommissionRewardsPool, commission_reward)
	rewards.RewardsPool.Add(rewards.RewardsPool, reward)

	self.ModifyValidatorRewards(validator_address, rewards)
}

type Field interface {
	ValidatorV1 | Validator | ValidatorInfo | ValidatorRewards
}

func (self *Validators) getFieldFor(t any) []byte {
	switch tt := t.(type) {
	case *ValidatorV1:
		return self.validator_field
	case *Validator:
		return self.validator_field
	case *ValidatorInfo:
		return self.validator_info_field
	case *ValidatorRewards:
		return self.validator_rewards_field
	default:
		err := fmt.Errorf("FieldByType: Unexpected type %T", tt)
		panic(err)
	}
}

func Get[T Field](v *Validators, validator_address *common.Address) (ret *T) {
	key := contract_storage.Stor_k_1(v.getFieldFor(ret), validator_address[:])
	v.storage.Get(key, func(bytes []byte) {
		ret = new(T)
		rlp.MustDecodeBytes(bytes, ret)
	})
	return
}

func Save[T Field](v *Validators, address *common.Address, data *T) {
	key := contract_storage.Stor_k_1(v.getFieldFor(data), address[:])
	v.storage.Put(key, rlp.MustEncodeToBytes(data))
}

func Modify[T Field](v *Validators, address *common.Address, data *T) {
	if data == nil {
		panic("Modify: data to modify cannot be nil")
	}

	// This could happen only due to some serious logic bug
	if v.ValidatorExists(address) == false {
		panic("Modify: non existent validator")
	}
	Save(v, address, data)
}
