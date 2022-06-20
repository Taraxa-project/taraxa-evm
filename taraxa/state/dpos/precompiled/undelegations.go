package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

type Undelegation struct {
	// Amount of TARA that accound should be able to get
	Amount *big.Int

	// Block number when the withdrawal be ready
	Block types.BlockNum

	// TODO: we will needed it for slashing
	Validator *common.Address
}

type Undelegations struct {
	storage *StorageWrapper
	// <delegator addres -> list of undelegations> as each delegator can undelegate multiple times
	undelegations_map map[common.Address]*IterableMap

	undelegations_field           []byte
	delegator_undelegations_field []byte
}

func (self *Undelegations) Init(stor *StorageWrapper, prefix []byte) {
	self.storage = stor

	// Init Delegations storage fields keys - relative to the prefix
	self.undelegations_field = append(prefix, []byte{0}...)
	self.delegator_undelegations_field = append(prefix, []byte{1}...)
}

// Returns true if for given values there is undelegation in queue
func (self *Undelegations) UndelegationExists(delegator_address *common.Address, validator_address *common.Address) bool {
	delegator_undelegations := self.getDelegatorUndelegationsList(delegator_address)
	return delegator_undelegations.AccountExists(validator_address)
}

// Returns undelegation object from queue
func (self *Undelegations) GetUndelegation(delegator_address *common.Address, validator_address *common.Address) (undelegation *Undelegation) {
	key := self.genUndelegationKey(delegator_address, validator_address)
	self.storage.Get(key, func(bytes []byte) {
		undelegation = new(Undelegation)
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
	undelegation.Validator = validator_address

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

// Returns all undelegations for given address
func (self *Undelegations) GetUndelegations(delegator_address *common.Address, batch uint32, count uint32) ([]common.Address, bool) {
	undelegations_list := self.getDelegatorUndelegationsList(delegator_address)
	return undelegations_list.GetAccounts(batch, count)
}

// Returns list of undelegations for given address
func (self *Undelegations) getDelegatorUndelegationsList(delegator_address *common.Address) *IterableMap {
	delegator_undelegations, found := self.undelegations_map[*delegator_address]
	if !found {
		delegator_undelegations = new(IterableMap)
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
	return stor_k_1(self.undelegations_field, validator_address[:], delegator_address[:])
}
