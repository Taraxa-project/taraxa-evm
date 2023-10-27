package contract_storage

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

type AddressesIMapReader struct {
	addresses IterableMapReader
}

// Inits iterable map with prefix, so multiple iterable maps can coexists thanks to different prefixes
func (self *AddressesIMapReader) Init(stor *StorageReaderWrapper, prefix []byte) {
	self.addresses.Init(stor, prefix)
}

// Checks is account exists in iterable map
func (self *AddressesIMapReader) AccountExists(account *common.Address) bool {
	return self.addresses.ItemExists(account.Bytes())
}

func (self *AddressesIMapReader) GetAccounts(batch uint32, count uint32) (result []common.Address, end bool) {
	items, end := self.addresses.GetItems(batch, count)

	result = make([]common.Address, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = common.BytesToAddress(items[idx])
	}

	return
}

// Returns number of stored items
func (self *AddressesIMapReader) GetCount() (count uint32) {
	return self.addresses.GetCount()
}

// AccountsIMap is a IterableMap wrapper for storing account addresses
type AddressesIMap struct {
	AddressesIMapReader
	addresses IterableMap
}

// Inits iterable map with prefix, so multiple iterable maps can coexists thanks to different prefixes
func (self *AddressesIMap) Init(stor *StorageWrapper, prefix []byte) {
	self.AddressesIMapReader.Init(&stor.StorageReaderWrapper, prefix)
	self.addresses.Init(stor, prefix)
}

// Creates account from iterable map
func (self *AddressesIMap) CreateAccount(account *common.Address) bool {
	return self.addresses.CreateItem(account.Bytes())
}

// Removes account from iterable map, returns number of left accounts in the iterable map
func (self *AddressesIMap) RemoveAccount(account *common.Address) uint32 {
	return self.addresses.RemoveItem(account.Bytes())
}
