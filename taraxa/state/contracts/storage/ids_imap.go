package contract_storage

import (
	"math/big"
)

type IdsIMapReader struct {
	ids IterableMapReader
}

// Inits iterable map with prefix, so multiple iterable maps can coexists thanks to different prefixes
func (self *IdsIMapReader) Init(stor *StorageReaderWrapper, prefix []byte) {
	self.ids.Init(stor, prefix)
}

// Checks is Id exists in iterable map
func (self *IdsIMapReader) IdExists(id *big.Int) bool {
	return self.ids.ItemExists(id.Bytes())
}

func (self *IdsIMapReader) GetIds(batch uint32, count uint32) (result []*big.Int, end bool) {
	items, end := self.ids.GetItems(batch*count, count)

	result = make([]*big.Int, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = new(big.Int).SetBytes(items[idx])
	}

	return
}

func (self *IdsIMap) GetAllIds() []*big.Int {
	items, _ := self.Ids.GetItems(0, self.Ids.GetCount())

	result := make([]*big.Int, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = new(big.Int).SetBytes(items[idx])
	}

	return result
}

// Returns number of stored items
func (self *IdsIMapReader) GetCount() (count uint32) {
	return self.ids.GetCount()
}

// IdsIMap is a IterableMap wrapper for storing Id Ids
type IdsIMap struct {
	IdsIMapReader
	Ids IterableMap
}

// Inits iterable map with prefix, so multiple iterable maps can coexists thanks to different prefixes
func (self *IdsIMap) Init(stor *StorageWrapper, prefix []byte) {
	self.IdsIMapReader.Init(&stor.StorageReaderWrapper, prefix)
	self.Ids.Init(stor, prefix)
}

// Creates Id from iterable map
func (self *IdsIMap) CreateId(id *big.Int) uint32 {
	return self.Ids.CreateItem(id.Bytes())
}

// Removes Id from iterable map, returns number of left Ids in the iterable map
func (self *IdsIMap) RemoveId(id *big.Int) uint32 {
	return self.Ids.RemoveItem(id.Bytes())
}
