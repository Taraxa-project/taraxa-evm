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
	ValidatorV1

	// Undelegation id
	// TODO: could be potenti
	Id *big.Int
}

type Undelegations struct {
	storage *contract_storage.StorageWrapper
	// <delegator address -> list of validators addresses> as each delegator can undelegate from multiple validators at the same time
	undelegations_map map[common.Address]*contract_storage.AddressesIMap

	// <delegator's validator address -> list of undelegations ids> as each delegator can have multiple undelegations from the same validator at the same time
	// Note: used for post corvus hardfork undelegations processing
	undelegations_ids_map map[common.Address]*contract_storage.IdsIMap

	undelegations_field               []byte
	delegator_undelegations_field     []byte
	delegator_undelegations_ids_field []byte
}

func (self *Undelegations) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.undelegations_field = append(prefix, []byte{0}...)
	self.delegator_undelegations_field = append(prefix, []byte{1}...)
	self.delegator_undelegations_ids_field = append(prefix, []byte{2}...)
}

// Returns true if for given values there is undelegation in queue
func (self *Undelegations) PreCornusHfUndelegationExists(delegator_address *common.Address, validator_address *common.Address) bool {
	delegator_undelegations := self.getDelegatorUndelegationsList(delegator_address)
	if !delegator_undelegations.AccountExists(validator_address) {
		return false
	}

	if self.GetPreCornusHfUndelegation(delegator_address, validator_address) == nil {
		return false
	}

	return true
}

// Returns undelegation object from queue
func (self *Undelegations) GetPreCornusHfUndelegation(delegator_address *common.Address, validator_address *common.Address) (undelegation *UndelegationV1) {
	key := self.genUndelegationKey(delegator_address, validator_address)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(UndelegationV1)
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
func (self *Undelegations) CreateUndelegation(delegator_address *common.Address, validator_address *common.Address, block types.BlockNum, amount *big.Int) {
	undelegation := new(Undelegation)
	undelegation.Amount = amount
	undelegation.Block = block

	undelegation_key := self.genUndelegationKey(delegator_address, validator_address)
	self.storage.Put(undelegation_key, rlp.MustEncodeToBytes(undelegation))

	undelegations_list := self.getDelegatorUndelegationsList(delegator_address)
	undelegations_list.CreateAccount(validator_address)
}

// Removes undelegation object from storage
func (self *Undelegations) RemoveUndelegation(delegator_address *common.Address, validator_address *common.Address) {
	undelegation_key := self.genUndelegationKey(delegator_address, validator_address)
	self.storage.Put(undelegation_key, nil)

	delegator_undelegations := self.getDelegatorUndelegationsList(delegator_address)

	if delegator_undelegations.RemoveAccount(validator_address) == 0 {
		self.removeDelegatorUndelegationList(delegator_address)
	}
}

// Returns all addressess of validators, from which is delegator <delegator_address> currently undelegating
func (self *Undelegations) GetDelegatorValidatorsAddresses(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	undelegations_list := self.getDelegatorUndelegationsList(delegator_address)
	return undelegations_list.GetAccounts(batch, count)
}

// Returns list of undelegations for given address
func (self *Undelegations) getDelegatorUndelegationsList(delegator_address *common.Address) *contract_storage.AddressesIMap {
	delegator_undelegations, found := self.undelegations_map[*delegator_address]
	if !found {
		delegator_undelegations = new(contract_storage.AddressesIMap)
		delegator_undelegations_tmp := append(self.delegator_undelegations_field, delegator_address[:]...)
		delegator_undelegations.Init(self.storage, delegator_undelegations_tmp)
	}

	return delegator_undelegations
}

// Removes undelefation from the list of undelegations
func (self *Undelegations) removeDelegatorUndelegationList(delegator_address *common.Address) {
	delete(self.undelegations_map, *delegator_address)
}

// Return key to storage where undelegations is stored
func (self *Undelegations) genUndelegationKey(delegator_address *common.Address, validator_address *common.Address) *common.Hash {
	return contract_storage.Stor_k_1(self.undelegations_field, validator_address[:], delegator_address[:])
}