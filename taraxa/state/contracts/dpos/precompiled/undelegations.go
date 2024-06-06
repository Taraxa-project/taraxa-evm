package dpos

import (
	"math/big"

	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

// Pre cornus hardfork - without undelegation Id
type UndelegationV1 struct {
	// Amount of TARA that accound should be able to get
	Amount *big.Int

	// Block number when the withdrawal be ready
	Block types.BlockNum
}

// Post cornus hardfork - with undelegation Id. Ids are used to support multiple undelegations at the same time
// TODO: rename to Undelegation
type UndelegationV2 struct {
	UndelegationV1

	// Undelegation id
	Id *big.Int
}

type DelegatorV2Undelegations struct {
	// list of validators addresses, from which delegator undelegated
	Validators *contract_storage.AddressesIMap
	// <validator address -> list of undelegations ids> as each delegator can have multiple undelegations from the same validator at the same time
	// Note: used for post corvus hardfork undelegations processing
	Undelegations_ids_map map[common.Address]*contract_storage.IdsIMap
}

type Undelegations struct {
	storage *contract_storage.StorageWrapper
	// V1 - pre cornus hardfork undelegations
	// <delegator address -> list of validators addresses> as each delegator can undelegate from multiple validators at the same time
	v1_undelegations_map map[common.Address]*contract_storage.AddressesIMap

	// V2 - post cornus hardfork undelegations
	// <delegator address -> DelegatorV2Undelegations object>
	v2_undelegations_map map[common.Address]DelegatorV2Undelegations

	undelegations_field                  []byte
	delegator_v1_undelegations_field     []byte
	delegator_v2_undelegations_field     []byte
	delegator_v2_undelegations_ids_field []byte
}

func (self *Undelegations) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.undelegations_field = append(prefix, []byte{0}...)
	self.delegator_v1_undelegations_field = append(prefix, []byte{1}...)
	self.delegator_v2_undelegations_field = append(prefix, []byte{2}...)
	self.delegator_v2_undelegations_ids_field = append(prefix, []byte{3}...)
}

// Returns true if for given values there is undelegation in queue
func (self *Undelegations) UndelegationV1Exists(delegator_address *common.Address, validator_address *common.Address) bool {
	return self.undelegationExists(delegator_address, validator_address, false)
}

func (self *Undelegations) UndelegationV2Exists(delegator_address *common.Address, validator_address *common.Address) bool {
	return self.undelegationExists(delegator_address, validator_address, true)
}

func (self *Undelegations) undelegationExists(delegator_address *common.Address, validator_address *common.Address, is_v2 bool) bool {
	delegator_undelegations := self.getDelegatorUndelegationsList(delegator_address, is_v2)
	return delegator_undelegations.AccountExists(validator_address)
}

// Returns undelegation object from queue
func (self *Undelegations) GetUndelegationV1(delegator_address *common.Address, validator_address *common.Address) (undelegation *UndelegationV1) {
	key := self.genUndelegationKey(delegator_address, validator_address, nil)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV1)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

func (self *Undelegations) GetUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) (undelegation *UndelegationV2) {
	key := self.genUndelegationKey(delegator_address, validator_address, undelegation_id)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV2)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

// Returns number of undelegations for specified address
func (self *Undelegations) GetUndelegationsCount(delegator_address *common.Address) uint32 {
	delegator_undelegations := self.getDelegatorUndelegationsList(delegator_address)
	return delegator_undelegations.GetCount()
}

// Creates undelegation object in storage
func (self *Undelegations) CreateUndelegationV1(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV1)
	undelegation.Amount = amount
	undelegation.Block = block

	self.createUndelegation(delegator_address, validator_address, nil, rlp.MustEncodeToBytes(undelegation))
}

func (self *Undelegations) CreateUndelegationV2(delegator_address *common.Address, validator_address *common.Address, delegator_nonce *big.Int, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV2)
	undelegation.Amount = amount
	undelegation.Block = block
	undelegation.Id = delegator_nonce

	self.createUndelegation(delegator_address, validator_address, delegator_nonce, rlp.MustEncodeToBytes(undelegation))
}

func (self *Undelegations) createUndelegation(delegator_address *common.Address, validator_address *common.Address, delegator_nonce *big.Int, undelegation_data []byte) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address, delegator_nonce)
	self.storage.Put(undelegation_key, undelegation_data)

	undelegations_list := self.getDelegatorUndelegationsList(delegator_address, true)
	undelegations_list.CreateAccount(validator_address)
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegationV1(delegator_address *common.Address, validator_address *common.Address) {
	self.removeUndelegation(delegator_address, validator_address, nil, false)
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) {
	self.removeUndelegation(delegator_address, validator_address, undelegation_id, true)
}

func (self *Undelegations) removeUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int, is_v2 bool) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address, undelegation_id)
	self.storage.Put(undelegation_key, nil)

	delegator_undelegations := self.getDelegatorUndelegationsList(delegator_address, is_v2)

	if delegator_undelegations.RemoveAccount(validator_address) == 0 {
		self.removeDelegatorUndelegationList(delegator_address, is_v2)
	}
}

// Returns all addressess of validators, from which is delegator <delegator_address> currently undelegating
func (self *Undelegations) GetDelegatorValidatorsAddresses(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	undelegations_list := self.getDelegatorUndelegationsList(delegator_address)
	return undelegations_list.GetAccounts(batch, count)
}

// Returns list of undelegations for given address
func (self *Undelegations) getDelegatorUndelegationsList(delegator_address *common.Address, is_v2 bool) *contract_storage.AddressesIMap {
	undelegations_map := self.getUndelegationsMap(is_v2) // returns reference to Undelegations struct member map
	delegator_undelegations, found := undelegations_map[*delegator_address]
	if !found {
		delegator_undelegations = new(contract_storage.AddressesIMap)
		var delegator_undelegations_prefix []byte
		if is_v2 {
			delegator_undelegations_prefix = append(self.delegator_v2_undelegations_field, delegator_address[:]...)
		} else {
			delegator_undelegations_prefix = append(self.delegator_v1_undelegations_field, delegator_address[:]...)
		}

		delegator_undelegations.Init(self.storage, delegator_undelegations_prefix)
	}

	return delegator_undelegations
}

// Removes undelefation from the list of undelegations
func (self *Undelegations) removeDelegatorUndelegationList(delegator_address *common.Address, is_v2 bool) {
	delete(self.getUndelegationsMap(is_v2), *delegator_address)
}

func (self *Undelegations) getUndelegationsMap(is_v2 bool) map[common.Address]*contract_storage.AddressesIMap {
	if is_v2 {
		return self.v2_undelegations_map
	}

	return self.v1_undelegations_map
}

// Return key to storage where undelegations is stored
func (self *Undelegations) genUndelegationKey(delegator_address *common.Address, validator_address *common.Address) *common.Hash {
	return contract_storage.Stor_k_1(self.undelegations_field, validator_address[:], delegator_address[:])
}
