package dpos

import (
	"log"
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

	// Undelegation id (unique per delegator address)
	Id uint64
}

type DelegatorV2Undelegations struct {
	// list of validators addresses, from which delegator undelegated
	Validators *contract_storage.AddressesIMap
	// <validator address -> list of undelegations ids> as each delegator can have multiple undelegations from the same validator at the same time
	// Note 1: used for post corvus hardfork undelegations processing
	// Note 2: Undelegations_ids_map should contain only validators addresses that are also in Validators struct member
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

	undelegations_field                            []byte
	delegator_v1_undelegations_field               []byte
	delegator_v2_undelegations_field               []byte
	delegator_v2_undelegations_ids_field           []byte
	delegator_v2_undelegations_last_uniqe_id_field []byte
}

func (self *Undelegations) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.undelegations_field = append(prefix, []byte{0}...)
	self.delegator_v1_undelegations_field = append(prefix, []byte{1}...)
	self.delegator_v2_undelegations_field = append(prefix, []byte{2}...)
	self.delegator_v2_undelegations_ids_field = append(prefix, []byte{3}...)
	self.delegator_v2_undelegations_last_uniqe_id_field = append(prefix, []byte{4}...)
}

// Returns true if for given values there is undelegation in queue
func (self *Undelegations) UndelegationExists(delegator_address *common.Address, validator_address *common.Address, undelegation_id *uint64) bool {
	if undelegation_id != nil {
		return self.undelegationV2Exists(delegator_address, validator_address, *undelegation_id)
	}

	return self.UndelegationV1Exists(delegator_address, validator_address)
}

func (self *Undelegations) UndelegationV1Exists(delegator_address *common.Address, validator_address *common.Address) bool {
	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	return validators_map.AccountExists(validator_address)
}

func (self *Undelegations) undelegationV2Exists(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) bool {
	_, ids_map := self.GetUndelegationsV2Maps(delegator_address, validator_address)
	return ids_map.IdExists(undelegation_id)
}

// Returns undelegation object from queue
func (self *Undelegations) GetUndelegationBaseObject(delegator_address *common.Address, validator_address *common.Address, undelegation_id *uint64) (undelegation *UndelegationV1) {
	if undelegation_id != nil {
		undelegation_v2 := self.GetUndelegationV2(delegator_address, validator_address, *undelegation_id)
		if undelegation_v2 != nil {
			undelegation = &undelegation_v2.UndelegationV1
		} else {
			undelegation = nil
		}
	} else {
		undelegation = self.GetUndelegationV1(delegator_address, validator_address)
	}

	return
}

func (self *Undelegations) GetUndelegationV1(delegator_address *common.Address, validator_address *common.Address) (undelegation *UndelegationV1) {
	key := self.genUndelegationV1Key(delegator_address, validator_address)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV1)
		rlp.MustDecodeBytes(bytes, undelegation)
	})

	return
}

func (self *Undelegations) GetUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) (undelegation *UndelegationV2) {
	key := self.genUndelegationV2Key(delegator_address, validator_address, undelegation_id)
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
	count := uint32(0)

	undelegations_v2_validators, _ := self.GetUndelegationsV2Maps(delegator_address, nil)
	for _, undelegations_v2_validator := range undelegations_v2_validators.GetAllAccounts() {
		_, undelegations_v2_ids := self.GetUndelegationsV2Maps(delegator_address, &undelegations_v2_validator)

		count += undelegations_v2_ids.GetCount()
	}

	return count
}

func (self *Undelegations) CreateUndelegationV1(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) {
	undelegation := new(UndelegationV1)
	undelegation.Amount = amount
	undelegation.Block = block
	self.saveUndelegationObject(self.genUndelegationV1Key(delegator_address, validator_address), rlp.MustEncodeToBytes(undelegation))
	log.Println("CreateUndelegation key: ", self.genUndelegationV1Key(delegator_address, validator_address).String())
	log.Println("CreateUndelegation value: ", rlp.MustEncodeToBytes(undelegation))
	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	validators_map.CreateAccount(validator_address)
}

func (self *Undelegations) CreateUndelegationV2(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) uint64 {
	undelegation := new(UndelegationV2)
	undelegation.Amount = amount
	undelegation.Block = block
	undelegation.Id = self.genUniqueId(delegator_address)

	self.saveUndelegationObject(self.genUndelegationV2Key(delegator_address, validator_address, undelegation.Id), rlp.MustEncodeToBytes(undelegation))

	validators_map, ids_map := self.GetUndelegationsV2Maps(delegator_address, validator_address)
	if ids_map.CreateId(undelegation.Id) == 1 {
		validators_map.CreateAccount(validator_address)
	}

	return undelegation.Id
}

func (self *Undelegations) genUniqueId(delegator_address *common.Address) uint64 {
	key := contract_storage.Stor_k_1(self.delegator_v2_undelegations_last_uniqe_id_field, delegator_address[:])

	unique_id := uint64(0)
	self.storage.Get(key, func(bytes []byte) {
		unique_id = contract_storage.BytesToUint64(bytes)
	})

	unique_id++
	self.storage.Put(key, contract_storage.Uint64ToBytes(unique_id))

	return unique_id
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegation(delegator_address *common.Address, validator_address *common.Address, undelegation_id *uint64) {
	if undelegation_id != nil {
		self.removeUndelegationV2(delegator_address, validator_address, *undelegation_id)
	} else {
		self.removeUndelegationV1(delegator_address, validator_address)
	}
}

func (self *Undelegations) removeUndelegationV1(delegator_address *common.Address, validator_address *common.Address) {
	self.removeUndelegationObject(self.genUndelegationV1Key(delegator_address, validator_address))

	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	if validators_map.RemoveAccount(validator_address) == 0 {
		delete(self.v1_undelegations_map, *delegator_address)
	}
}

func (self *Undelegations) removeUndelegationV2(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) {
	self.removeUndelegationObject(self.genUndelegationV2Key(delegator_address, validator_address, undelegation_id))

	validators_map, ids_map := self.GetUndelegationsV2Maps(delegator_address, validator_address)
	if ids_map.RemoveId(undelegation_id) == 0 {
		if validators_map.RemoveAccount(validator_address) == 0 {
			delete(self.v2_undelegations_map, *delegator_address)
		} else {
			delete(self.v2_undelegations_map[*delegator_address].Undelegations_ids_map, *validator_address)
		}
	}
}

func (self *Undelegations) saveUndelegationObject(key *common.Hash, undelegation_data []byte) {
	self.storage.Put(key, undelegation_data)
}

func (self *Undelegations) removeUndelegationObject(key *common.Hash) {
	self.storage.Put(key, nil)
}

// Returns all addressess of validators, from which is delegator <delegator_address> currently undelegating v1
func (self *Undelegations) GetUndelegationsV1Validators(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	validators_map := self.getUndelegationsV1ValidatorsMap(delegator_address)
	return validators_map.GetAccounts(batch, count)
}

// Returns validator by idx, from which is delegator <delegator_address> currently undelegating v2
func (self *Undelegations) GetUndelegationsV2Validator(delegator_address *common.Address, validator_idx uint32) (*common.Address, bool) {
	validators_map, _ := self.GetUndelegationsV2Maps(delegator_address, nil)
	validators, end := validators_map.GetAccounts(validator_idx, 1)
	if len(validators) > 0 {
		return &validators[0], end
	}

	return nil, end
}

// Returns list of undelegations for given address
func (self *Undelegations) getUndelegationsV1ValidatorsMap(delegator_address *common.Address) *contract_storage.AddressesIMap {
	v1_undelegations_validators, found := self.v1_undelegations_map[*delegator_address]
	if !found {
		v1_undelegations_validators_prefix := append(self.delegator_v1_undelegations_field, delegator_address[:]...)

		v1_undelegations_validators = new(contract_storage.AddressesIMap)
		v1_undelegations_validators.Init(self.storage, v1_undelegations_validators_prefix)
	}

	return v1_undelegations_validators
}

func (self *Undelegations) GetUndelegationsV2Maps(delegator_address *common.Address, validator_address *common.Address) (validators_map *contract_storage.AddressesIMap, ids_map *contract_storage.IdsIMap) {
	v2_undelegations_validators, found := self.v2_undelegations_map[*delegator_address]
	if !found {
		v2_undelegations_validators = new(DelegatorV2Undelegations)

		v2_undelegations_validators_prefix := append(self.delegator_v2_undelegations_field, delegator_address[:]...)
		v2_undelegations_validators.Validators = new(contract_storage.AddressesIMap)
		v2_undelegations_validators.Validators.Init(self.storage, v2_undelegations_validators_prefix)
	}
	validators_map = v2_undelegations_validators.Validators

	if validator_address != nil {
		v2_undelegations_ids, ids_found := v2_undelegations_validators.Undelegations_ids_map[*validator_address]
		if !ids_found {
			v2_undelegations_ids_prefix := append(append(self.delegator_v2_undelegations_ids_field, delegator_address[:]...), validator_address[:]...)
			v2_undelegations_ids = new(contract_storage.IdsIMap)
			v2_undelegations_ids.Init(self.storage, v2_undelegations_ids_prefix)
		}

		ids_map = v2_undelegations_ids
	}

	return
}

// Return key to storage where undelegations V1 is stored
func (self *Undelegations) genUndelegationV1Key(delegator_address *common.Address, validator_address *common.Address) *common.Hash {
	// Pre-cornus hf undelegation key is created from validator & delegator address
	return contract_storage.Stor_k_1(self.undelegations_field, validator_address[:], delegator_address[:])
}

// Return key to storage where undelegations V2 is stored
func (self *Undelegations) genUndelegationV2Key(delegator_address *common.Address, validator_address *common.Address, undelegation_id uint64) *common.Hash {
	// Post-cornus hf undelegation key is created from delegator address, validator address & and undelegation id
	return contract_storage.Stor_k_1(self.undelegations_field, delegator_address[:], validator_address[:], contract_storage.Uint64ToBytes(undelegation_id))
}
