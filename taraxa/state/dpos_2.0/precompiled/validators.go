package dpos_2

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
	storage         StorageWrapper
	validators_list IterableMap

	validators_field      []byte
	validators_info_field []byte
}

func (self *Validators) Init(stor StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Validators storage fields keys - relative to the prefix
	self.validators_field = append(prefix, []byte{0}...)
	self.validators_info_field = append(prefix, []byte{1}...)
	validators_list_field := append(prefix, []byte{2}...)

	self.validators_list.Init(self.storage, validators_list_field)
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
	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("ModifyValidator: non existent validator")
	}

	// TODO: IMPORTANT -> is there an reason why to use stor_k_2 vs stor_k_1 ???
	key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(key, rlp.MustEncodeToBytes(validator))
}

func (self *Validators) CreateValidator(validator_address *common.Address, block types.BlockNum, stake *big.Int, commission uint16, description string, endpoint string) {
	// Creates Validator object in storage
	validator := new(Validator)
	validator.CommissionRewardsPool = bigutil.Big0
	validator.RewardsPool = bigutil.Big0
	validator.Commission = commission
	validator.TotalStake = stake
	validator.LastUpdated = block

	// TODO: IMPORTANT -> is there an reason why to use stor_k_2 vs stor_k_1 ???
	validator_key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(validator_key, rlp.MustEncodeToBytes(validator))

	// Creates Validator_info object in storage
	validator_info := new(ValidatorInfo)
	validator_info.Description = description
	validator_info.Endpoint = endpoint

	// TODO: IMPORTANT -> is there an reason why to use stor_k_2 vs stor_k_1 ???
	validator_info_key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Put(validator_info_key, rlp.MustEncodeToBytes(validator_info))

	// Adds validator into the list of all validators
	self.validators_list.CreateAccount(validator_address)
}

func (self *Validators) DeleteValidator(validator_address *common.Address) {
	validator_key := stor_k_1(self.validators_field, validator_address[:])
	self.storage.Put(validator_key, nil)

	validator_info_key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Put(validator_info_key, nil)

	// Removes validator into the list of all validators
	self.validators_list.RemoveAccount(validator_address)
}

func (self *Validators) GetValidatorInfo(validator_address *common.Address) (validator_info *ValidatorInfo) {
	// TODO: IMPORTANT -> is there an reason why to use stor_k_2 vs stor_k_1 ???
	key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Get(key, func(bytes []byte) {
		validator_info = new(ValidatorInfo)
		rlp.MustDecodeBytes(bytes, validator_info)
	})

	return
}

func (self *Validators) ModifyValidatorInfo(validator_address *common.Address, validator_info *ValidatorInfo) {
	// This could happen only due to some serious logic bug
	if self.ValidatorExists(validator_address) == false {
		panic("ModifyValidatorInfo: non existent validator")
	}

	// TODO: IMPORTANT -> is there an reason why to use stor_k_2 vs stor_k_1 ???
	key := stor_k_1(self.validators_info_field, validator_address[:])
	self.storage.Put(key, rlp.MustEncodeToBytes(validator_info))
}

func (self *Validators) GetValidatorsAddresses(batch uint32, count uint32) (result []common.Address, end bool) {
	return self.validators_list.GetAccounts(batch, count)
}
