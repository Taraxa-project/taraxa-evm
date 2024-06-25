package dpos

import (
	"fmt"
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
type UndelegationV2 struct {
	UndelegationV1

	// Undelegation id
	Id *big.Int
}

type Undelegations struct {
	storage *contract_storage.StorageWrapper
	// V1 - pre cornus hardfork undelegations
	// <delegator address -> list of validators addresses> as each delegator can undelegate from multiple validators at the same time
	v1_undelegations_map map[common.Address]*contract_storage.AddressesIMap

	// V2 - post cornus hardfork undelegations
	// <delegator address -> list of undelegations ids
	v2_undelegations_map map[common.Address]*contract_storage.IdsIMap

	undelegations_field              []byte
	delegator_v1_undelegations_field []byte
	delegator_v2_undelegations_field []byte
}

func (self *Undelegations) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.undelegations_field = append(prefix, []byte{0}...)
	self.delegator_v1_undelegations_field = append(prefix, []byte{1}...)
	self.delegator_v2_undelegations_field = append(prefix, []byte{2}...)
}

// Returns true if for given values there is undelegation in queue
func (self *Undelegations) UndelegationExists(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) bool {
	if undelegation_id != nil {
		return self.UndelegationV2Exists(delegator_address, undelegation_id)
	} else if validator_address != nil {
		return self.UndelegationV1Exists(delegator_address, validator_address)
	}

	fmt.Println("UndelegationExists called with both validator_address & undelegation_id == nil")
	return false
}

func (self *Undelegations) UndelegationV1Exists(delegator_address *common.Address, validator_address *common.Address) bool {
	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	return validators_map.AccountExists(validator_address)
}

func (self *Undelegations) UndelegationV2Exists(delegator_address *common.Address, undelegation_id *big.Int) bool {
	ids_map := self.getUndelegationsV2IdsMap(delegator_address)
	return ids_map.IdExists(undelegation_id)
}

// Returns undelegation object from queue
func (self *Undelegations) GetUndelegationBaseObject(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) (undelegation *UndelegationV1) {
	if undelegation_id != nil {
		undelegation_v2 := self.GetUndelegationV2(delegator_address, undelegation_id)
		if undelegation_v2 != nil {
			undelegation = &undelegation_v2.UndelegationV1
		} else {
			undelegation = nil
		}
	} else if validator_address != nil {
		undelegation = self.GetUndelegationV1(delegator_address, validator_address)
	} else {
		fmt.Println("GetUndelegationBaseObject called with both validator_address & undelegation_id == nil")
	}

	return
}

func (self *Undelegations) GetUndelegationV1(delegator_address *common.Address, validator_address *common.Address) (undelegation *UndelegationV1) {
	key := self.genUndelegationKey(delegator_address, validator_address, nil)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV1)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

func (self *Undelegations) GetUndelegationV2(delegator_address *common.Address, undelegation_id *big.Int) (undelegation *UndelegationV2) {
	key := self.genUndelegationKey(delegator_address, nil, undelegation_id)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV2)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

// Returns number of V1 undelegations for specified address
func (self *Undelegations) GetUndelegationsV1Count(delegator_address *common.Address) uint32 {
	return self.getUndelegationsV1ValidatorsMap(delegator_address).GetCount()
}

// Returns number of V2 undelegations for specified address
func (self *Undelegations) GetUndelegationsV2Count(delegator_address *common.Address) uint32 {
	return self.getUndelegationsV2IdsMap(delegator_address).GetCount()
}

// Creates undelegation object in storage
func (self *Undelegations) CreateUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int, block types.BlockNum, amount *big.Int) {
	if undelegation_id != nil {
		// TODO: must save validator address too !!!
		self.CreateUndelegationV2(delegator_address, undelegation_id, block, amount)
	} else if validator_address != nil {
		self.CreateUndelegationV1(delegator_address, validator_address, block, amount)
	} else {
		fmt.Println("CreateUndelegation called with both validator_address & undelegation_id == nil")
	}
}

func (self *Undelegations) CreateUndelegationV1(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV1)
	undelegation.Amount = amount
	undelegation.Block = block

	self.saveUndelegationObject(delegator_address, validator_address, nil, rlp.MustEncodeToBytes(undelegation))

	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	validators_map.CreateAccount(validator_address)
}

func (self *Undelegations) CreateUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV2)
	undelegation.Amount = amount
	undelegation.Block = block
	undelegation.Id = undelegation_id

	self.saveUndelegationObject(delegator_address, nil, undelegation_id, rlp.MustEncodeToBytes(undelegation))

	ids_map := self.getUndelegationsV2IdsMap(delegator_address)
	ids_map.CreateId(undelegation_id)
}

func (self *Undelegations) saveUndelegationObject(delegator_address *common.Address, validator_address *common.Address, delegator_nonce *big.Int, undelegation_data []byte) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address, delegator_nonce)
	self.storage.Put(undelegation_key, undelegation_data)
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) {
	if undelegation_id != nil {
		self.removeUndelegationV2(delegator_address, undelegation_id)
	} else if validator_address != nil {
		self.removeUndelegationV1(delegator_address, validator_address)
	} else {
		fmt.Println("RemoveUndelegation called with both validator_address & undelegation_id == nil")
	}
}

func (self *Undelegations) removeUndelegationV1(delegator_address *common.Address, validator_address *common.Address) {
	self.removeUndelegationObject(delegator_address, validator_address, nil)

	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	if validators_map.RemoveAccount(validator_address) == 0 {
		delete(self.v1_undelegations_map, *delegator_address)
	}
}

func (self *Undelegations) removeUndelegationV2(delegator_address *common.Address, undelegation_id *big.Int) {
	self.removeUndelegationObject(delegator_address, nil, undelegation_id)

	ids_map := self.getUndelegationsV2IdsMap(delegator_address)
	if ids_map.RemoveId(undelegation_id) == 0 {
		delete(self.v2_undelegations_map, *delegator_address)
	}
}

func (self *Undelegations) removeUndelegationObject(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address, undelegation_id)
	self.storage.Put(undelegation_key, nil)
}

// Returns all addressess of validators, from which is delegator <delegator_address> currently undelegating v1
func (self *Undelegations) GetUndelegationsV1Validators(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	return validators_map.GetAccounts(batch, count)
}

// Returns all addressess of validators, from which is delegator <delegator_address> currently undelegating v1
func (self *Undelegations) GetUndelegationsV2Ids(delegator_address *common.Address, batch uint32, count uint32) ([]*big.Int, bool) {
	ids_map := self.getUndelegationsV2IdsMap(delegator_address)
	return ids_map.GetIds(batch, count)
}

// Returns list of undelegations for given address
func (self *Undelegations) getUndelegationsV1ValidatorsMap(delegator_address *common.Address) *contract_storage.AddressesIMap {
	v1_undelegations_validators, found := self.v1_undelegations_map[*delegator_address]
	if !found {
		v1_undelegations_prefix := append(self.delegator_v1_undelegations_field, delegator_address[:]...)

		v1_undelegations_validators = new(contract_storage.AddressesIMap)
		v1_undelegations_validators.Init(self.storage, v1_undelegations_prefix)
	}

	return v1_undelegations_validators
}

func (self *Undelegations) getUndelegationsV2IdsMap(delegator_address *common.Address) *contract_storage.IdsIMap {
	v2_undelegations_ids, found := self.v2_undelegations_map[*delegator_address]
	if !found {
		v2_undelegations_prefix := append(self.delegator_v2_undelegations_field, delegator_address[:]...)

		v2_undelegations_ids = new(contract_storage.IdsIMap)
		v2_undelegations_ids.Init(self.storage, v2_undelegations_prefix)
	}

	return v2_undelegations_ids
}

// Return key to storage where undelegations is stored
func (self *Undelegations) genUndelegationKey(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) *common.Hash {
	if undelegation_id != nil {
		// Post-cornus hf undelegation key is created from delegator address & undelegation id
		return contract_storage.Stor_k_1(self.undelegations_field, delegator_address[:], undelegation_id.Bytes())
	} else if validator_address != nil {
		// Pre-cornus hf undelegation key was created just from validator & delegator address
		return contract_storage.Stor_k_1(self.undelegations_field, validator_address[:], delegator_address[:])
	}

	fmt.Println("genUndelegationKey called with both validator_address & undelegation_id == nil")
	return &common.ZeroHash
}
