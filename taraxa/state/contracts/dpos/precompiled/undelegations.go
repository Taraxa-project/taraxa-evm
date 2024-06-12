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
	v2_undelegations_map map[common.Address]*DelegatorV2Undelegations

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
func (self *Undelegations) UndelegationExists(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) bool {
	if undelegation_id != nil {
		return self.undelegationV2Exists(delegator_address, validator_address, undelegation_id)
	}

	return self.undelegationV1Exists(delegator_address, validator_address)
}

func (self *Undelegations) undelegationV1Exists(delegator_address *common.Address, validator_address *common.Address) bool {
	validators_map := self.getUndelegationsV1Map(delegator_address)
	return validators_map.AccountExists(validator_address)
}

func (self *Undelegations) undelegationV2Exists(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) bool {
	_, ids_map := self.getUndelegationsV2Maps(delegator_address, validator_address)
	return ids_map.IdExists(undelegation_id)
}

// Returns undelegation object from queue
func (self *Undelegations) GetUndelegationBaseObject(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) (undelegation *UndelegationV1) {
	if undelegation_id != nil {
		undelegation_v2 := self.getUndelegationV2(delegator_address, validator_address, undelegation_id)
		undelegation = &undelegation_v2.UndelegationV1
	} else {
		undelegation = self.getUndelegationV1(delegator_address, validator_address)
	}

	return
}

func (self *Undelegations) getUndelegationV1(delegator_address *common.Address, validator_address *common.Address) (undelegation *UndelegationV1) {
	key := self.genUndelegationKey(delegator_address, validator_address, nil)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV1)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

func (self *Undelegations) getUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) (undelegation *UndelegationV2) {
	key := self.genUndelegationKey(delegator_address, validator_address, undelegation_id)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV2)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

// Returns number of undelegations for specified address
// TODO: fix
func (self *Undelegations) GetUndelegationsCount(delegator_address *common.Address) uint32 {
	validators_map := self.getUndelegationsV1Map(delegator_address)
	return validators_map.GetCount(1)
}

// Creates undelegation object in storage
func (self *Undelegations) CreateUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int, block types.BlockNum, amount *big.Int) {
	if undelegation_id != nil {
		self.createUndelegationV2(delegator_address, validator_address, undelegation_id, block, amount)
	} else {
		self.createUndelegationV1(delegator_address, validator_address, block, amount)
	}
}

func (self *Undelegations) createUndelegationV1(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV1)
	undelegation.Amount = amount
	undelegation.Block = block

	self.saveUndelegationObject(delegator_address, validator_address, nil, rlp.MustEncodeToBytes(undelegation))

	validators_map := self.getUndelegationsV1Map(delegator_address)
	validators_map.CreateAccount(validator_address)
}

func (self *Undelegations) createUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV2)
	undelegation.Amount = amount
	undelegation.Block = block
	undelegation.Id = undelegation_id

	self.saveUndelegationObject(delegator_address, validator_address, undelegation_id, rlp.MustEncodeToBytes(undelegation))

	validators_map, ids_map := self.getUndelegationsV2Maps(delegator_address, validator_address)
	if ids_map.CreateId(undelegation_id) == 1 {
		validators_map.CreateAccount(validator_address)
	}
}

func (self *Undelegations) saveUndelegationObject(delegator_address *common.Address, validator_address *common.Address, delegator_nonce *big.Int, undelegation_data []byte) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address, delegator_nonce)
	self.storage.Put(undelegation_key, undelegation_data)
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) {
	if undelegation_id != nil {
		self.removeUndelegationV2(delegator_address, validator_address, undelegation_id)
	} else {
		self.removeUndelegationV1(delegator_address, validator_address)
	}
}

func (self *Undelegations) removeUndelegationV1(delegator_address *common.Address, validator_address *common.Address) {
	self.removeUndelegationObject(delegator_address, validator_address, nil)

	validators_map := self.getUndelegationsV1Map(delegator_address)
	if validators_map.RemoveAccount(validator_address) == 0 {
		delete(self.v1_undelegations_map, *delegator_address)
	}
}

func (self *Undelegations) removeUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) {
	self.removeUndelegationObject(delegator_address, validator_address, undelegation_id)

	validators_map, ids_map := self.getUndelegationsV2Maps(delegator_address, validator_address)
	if ids_map.RemoveId(undelegation_id) == 0 {
		if validators_map.RemoveAccount(validator_address) == 0 {
			delete(self.v2_undelegations_map, *delegator_address)
		} else {
			delete(self.v2_undelegations_map[*delegator_address].Undelegations_ids_map, *validator_address)
		}
	}
}

func (self *Undelegations) removeUndelegationObject(delegator_address *common.Address, validator_address *common.Address, undelegation_id *big.Int) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address, undelegation_id)
	self.storage.Put(undelegation_key, nil)
}

// Returns all addressess of validators, from which is delegator <delegator_address> currently undelegating
func (self *Undelegations) GetDelegatorValidatorsAddresses(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	validators_map := self.getUndelegationsV1Map(delegator_address)
	return validators_map.GetAccounts(batch, count)
}

// Returns list of undelegations for given address
func (self *Undelegations) getUndelegationsV1Map(delegator_address *common.Address) *contract_storage.AddressesIMap {
	delegator_v1_undelegations, found := self.v1_undelegations_map[*delegator_address]
	if !found {
		delegator_v1_undelegations_prefix := append(self.delegator_v1_undelegations_field, delegator_address[:]...)

		delegator_v1_undelegations = new(contract_storage.AddressesIMap)
		delegator_v1_undelegations.Init(self.storage, delegator_v1_undelegations_prefix)
	}

	return delegator_v1_undelegations
}

func (self *Undelegations) getUndelegationsV2Maps(delegator_address *common.Address, validator_address *common.Address) (validators_map *contract_storage.AddressesIMap, ids_map *contract_storage.IdsIMap) {
	delegator_v2_undelegations, found := self.v2_undelegations_map[*delegator_address]
	if !found {
		delegator_v2_undelegations = new(DelegatorV2Undelegations)

		delegator_v2_undelegations_prefix := append(self.delegator_v2_undelegations_field, delegator_address[:]...)
		delegator_v2_undelegations.Validators = new(contract_storage.AddressesIMap)
		delegator_v2_undelegations.Validators.Init(self.storage, delegator_v2_undelegations_prefix)
	}

	v2_undelegations_ids, ids_found := delegator_v2_undelegations.Undelegations_ids_map[*validator_address]
	if !ids_found {
		v2_undelegations_ids_prefix := append(append(self.delegator_v2_undelegations_ids_field, delegator_address[:]...), validator_address[:]...)
		v2_undelegations_ids = new(contract_storage.IdsIMap)
		v2_undelegations_ids.Init(self.storage, v2_undelegations_ids_prefix)
	}

	validators_map = delegator_v2_undelegations.Validators
	ids_map = v2_undelegations_ids

	return
}

// Return key to storage where undelegations is stored
func (self *Undelegations) genUndelegationKey(delegator_address *common.Address, validator_address *common.Address) *common.Hash {
	return contract_storage.Stor_k_1(self.undelegations_field, validator_address[:], delegator_address[:])
}
