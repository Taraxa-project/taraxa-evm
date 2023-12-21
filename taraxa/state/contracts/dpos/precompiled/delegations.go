package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type Delegation struct {
	// Num of delegated tokens == delegator's stake
	Stake *big.Int

	// Block number related to rewards
	LastUpdated types.BlockNum
}

// Delegations type groups together all functionality related to creating/deleting/modifying/etc... delegations
// as such info is stored under multiple independent storage keys, it is important that caller does not need to
// think about all implementation details, but just calls functions on Delegations type
type Delegations struct {
	storage *contract_storage.StorageWrapper
	// <delegator addres -> list of validators> as each delegator can delegate to multiple validators
	delegators_validators map[common.Address]*contract_storage.AddressesIMap

	delegations_field                  []byte
	delegators_validators_field_prefix []byte
}

func (self *Delegations) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.delegations_field = append(prefix, []byte{0}...)
	self.delegators_validators_field_prefix = append(prefix, []byte{1}...)
}

// Checks if delegation exists
func (self *Delegations) DelegationExists(delegator_address *common.Address, validator_address *common.Address) bool {
	delegator_validators := self.getDelegatorValidatorsList(delegator_address)
	return delegator_validators.AccountExists(validator_address)
}

// Returns number of delegations for specified address
func (self *Delegations) GetDelegationsCount(delegator_address *common.Address) uint32 {
	delegator_validators := self.getDelegatorValidatorsList(delegator_address)
	return delegator_validators.GetCount()
}

// Gets delegation
func (self *Delegations) GetDelegation(delegator_address *common.Address, validator_address *common.Address) (delegation *Delegation) {
	key := self.genDelegationKey(delegator_address, validator_address)
	self.storage.Get(&key, func(bytes []byte) {
		delegation = new(Delegation)
		rlp.MustDecodeBytes(bytes, delegation)
	})

	return
}

func (self *Delegations) ModifyDelegation(delegator_address *common.Address, validator_address *common.Address, delegation *Delegation) {
	if delegation == nil {
		panic("ModifyDelegation: delegation cannot be nil")
	}

	delegator_validators := self.getDelegatorValidatorsList(delegator_address)

	// This could happen only due to some serious logic bug
	if delegator_validators.AccountExists(validator_address) == false {
		panic("ModifyDelegation: non existent delegation")
	}

	key := self.genDelegationKey(delegator_address, validator_address)
	self.storage.Put(&key, rlp.MustEncodeToBytes(delegation))
}

func (self *Delegations) CreateDelegation(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, stake *big.Int) {
	// Creates Delegation object in storage
	delegation := new(Delegation)
	delegation.Stake = stake
	delegation.LastUpdated = block

	delegation_key := self.genDelegationKey(delegator_address, validator_address)
	self.storage.Put(&delegation_key, rlp.MustEncodeToBytes(delegation))

	// Adds validator into delegator's validators list
	delegator_validators := self.getDelegatorValidatorsList(delegator_address)
	delegator_validators.CreateAccount(validator_address)
}

func (self *Delegations) RemoveDelegation(delegator_address *common.Address, validator_address *common.Address) {
	delegation_key := self.genDelegationKey(delegator_address, validator_address)
	self.storage.Put(&delegation_key, nil)

	// Removes validator from delegator's validators list
	delegator_validators := self.getDelegatorValidatorsList(delegator_address)
	remaining_validators_count := delegator_validators.RemoveAccount(validator_address)

	// If delegator does not delegate to anyone else, remove also his validators list
	if remaining_validators_count == 0 {
		self.removeDelegatorValidatorsList(delegator_address)
	}
}

func (self *Delegations) GetDelegatorValidatorsAddresses(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	delegator_validators := self.getDelegatorValidatorsList(delegator_address)
	return delegator_validators.GetAccounts(batch, count)
}

func (self *Delegations) getDelegatorValidatorsList(delegator_address *common.Address) *contract_storage.AddressesIMap {
	delegator_validators, found := self.delegators_validators[*delegator_address]
	if found == false {
		delegator_validators = new(contract_storage.AddressesIMap)
		delegator_validators_field := append(self.delegators_validators_field_prefix, delegator_address[:]...)
		delegator_validators.Init(self.storage, delegator_validators_field)
	}

	return delegator_validators
}

func (self *Delegations) GetAllDelegatorValidatorsAddresses(delegator_address *common.Address) []common.Address {
	delegator_validators := self.getDelegatorValidatorsList(delegator_address)
	return delegator_validators.GetAllAccounts()
}

func (self *Delegations) removeDelegatorValidatorsList(delegator_address *common.Address) {
	delete(self.delegators_validators, *delegator_address)
}

func (self *Delegations) genDelegationKey(delegator_address *common.Address, validator_address *common.Address) common.Hash {
	return contract_storage.Stor_k_2(self.delegations_field, validator_address[:], delegator_address[:])
}
