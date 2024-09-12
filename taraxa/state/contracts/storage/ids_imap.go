package contract_storage

type IdsIMapReader struct {
	ids IterableMapReader
}

// Inits iterable map with prefix, so multiple iterable maps can coexists thanks to different prefixes
func (self *IdsIMapReader) Init(stor *StorageReaderWrapper, prefix []byte) {
	self.ids.Init(stor, prefix)
}

// Checks is Id exists in iterable map
func (self *IdsIMapReader) IdExists(id uint64) bool {
	return self.ids.ItemExists(Uint64ToBytes(id))
}

func (self *IdsIMapReader) GetIdsFromIdx(start_idx uint32, count uint32) (result []uint64, end bool) {
	items, end := self.ids.GetItems(start_idx, count)

	result = make([]uint64, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = BytesToUint64(items[idx])
	}

	return
}

func (self *IdsIMapReader) GetAllIds() []uint64 {
	items, _ := self.ids.GetItems(0, self.ids.GetCount())

	result := make([]uint64, len(items))
	for idx := 0; idx < len(items); idx++ {
		result[idx] = BytesToUint64(items[idx])
	}

	return result
}

// Returns number of stored items
func (self *IdsIMapReader) GetCount() (count uint32) {
	return self.ids.GetCount()
}

// IdsIMap is a IterableMap wrapper for storing unique Ids
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
func (self *IdsIMap) CreateId(id uint64) uint32 {
	return self.Ids.CreateItem(Uint64ToBytes(id))
}

// Removes Id from iterable map, returns number of left Ids in the iterable map
func (self *IdsIMap) RemoveId(id uint64) uint32 {
	return self.Ids.RemoveItem(Uint64ToBytes(id))
}

func Uint64ToBytes(val uint64) []byte {
	r := make([]byte, 8)
	for i := uint64(0); i < 8; i++ {
		r[i] = byte((val >> (8 * i)) & 0xff)
	}
	return r
}

func BytesToUint64(val []byte) uint64 {
	r := uint64(0)
	for i := uint64(0); i < 8; i++ {
		r |= uint64(val[i]) << (8 * i)
	}
	return r
}
