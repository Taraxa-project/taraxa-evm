package contract_storage

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

// AccountsIMap is a IterableMap wrapper for storing account addresses
type AddressesIMap struct {
	addresses IterableMap
}

// Inits iterbale map with prefix, so multiple iterbale maps can coexists thanks to different prefixes
func (self *AddressesIMap) Init(stor *StorageWrapper, prefix []byte) {
	self.addresses.Init(stor, prefix)
}

// Checks is account exists in iterable map
func (self *AddressesIMap) AccountExists(account *common.Address) bool {
	return self.addresses.ItemExists(account.Bytes())
}

// Creates account from iterable map
func (self *AddressesIMap) CreateAccount(account *common.Address) bool {
	return self.addresses.CreateItem(account.Bytes())
}

// Removes account from iterable map, returns number of left accounts in the iterbale map
func (self *AddressesIMap) RemoveAccount(account *common.Address) uint32 {
	return self.addresses.RemoveItem(account.Bytes())
}

func (self *AddressesIMap) GetAccounts(batch uint32, count uint32) (result []common.Address, end bool) {
	items, end := self.addresses.GetItems(batch, count)

	result = make([]common.Address, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = common.BytesToAddress(items[idx])
	}

	return
}

func (self *AddressesIMap) GetAllAccounts() []common.Address {
	items, _ := self.addresses.GetItems(0, self.addresses.GetCount())

	result := make([]common.Address, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = common.BytesToAddress(items[idx])
	}

	return result
}

// Returns number of stored items
func (self *AddressesIMap) GetCount() (count uint32) {
	return self.addresses.GetCount()
}
