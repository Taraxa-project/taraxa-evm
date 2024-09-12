package contract_storage

import (
	"log"

	"github.com/Taraxa-project/taraxa-evm/common"
)

// IterableMap storage fields keys - relative to the prefix from Init function
var (
	field_items       = []byte{0}
	field_items_count = []byte{1}
	field_items_pos   = []byte{2}
)

type IterableMapReader struct {
	storage                  *StorageReaderWrapper
	items_storage_prefix     []byte       // items are stored under "items_storage_prefix + pos" key
	items_count_storage_key  *common.Hash // items count is stored under items_count_storage_key
	items_pos_storage_prefix []byte       // items positions are stored under "items_pos_storage_prefix + item" key
}

// Inits iterable map with prefix, so multiple iterbale maps can coexists thanks to different prefixes
func (self *IterableMapReader) Init(stor *StorageReaderWrapper, prefix []byte) {
	self.storage = stor
	self.items_storage_prefix = append(prefix, field_items...)
	self.items_count_storage_key = Stor_k_1(prefix, field_items_count)
	self.items_pos_storage_prefix = append(prefix, field_items_pos...)
}

// Checks is item exists in iterable map
func (self *IterableMapReader) ItemExists(item []byte) bool {
	item_exists, _ := self.itemExists(item)
	return item_exists
}

func (self *IterableMapReader) GetItems(start_idx uint32, count uint32) (result [][]byte, end bool) {
	// Gets items count
	items_count := self.GetCount()

	// No items in iterable map
	if items_count == 0 {
		end = true
		return
	}

	end_idx := start_idx + count

	// Invalid batch provided - there is not so many items in iterbale map
	if start_idx >= items_count {
		end = true
		return
	}

	if items_count <= end_idx {
		result = make([][]byte, items_count-start_idx)
		end = true
	} else {
		result = make([][]byte, count)
		end = false
	}

	// Start with index == 1, there is nothing saved on index == 0 as it is reserved to indicate non-existent item
	for idx := uint32(start_idx + 1); idx <= end_idx && idx <= items_count; idx++ {
		items_k := Stor_k_1(self.items_storage_prefix, uint32ToBytes(idx))

		var item []byte
		self.storage.Get(items_k, func(bytes []byte) {
			item = bytes
		})

		if len(item) == 0 {
			// This should never happen
			panic("Unable to find item " + string(item))
		}

		result[idx-start_idx-1] = item
	}

	return
}

// Returns number of stored items
func (self *IterableMapReader) GetCount() (count uint32) {
	count = 0
	self.storage.Get(self.items_count_storage_key, func(bytes []byte) {
		count = bytesToUint32(bytes)
	})

	return
}

// Checks is item exists in iterable map
// If item exists <true, position> it returned, otheriwse <false, 0>
func (self *IterableMapReader) itemExists(item []byte) (item_exists bool, item_pos uint32) {
	pos_k := Stor_k_1(self.items_pos_storage_prefix, item[:])
	item_pos = 0

	self.storage.Get(pos_k, func(bytes []byte) {
		item_pos = bytesToUint32(bytes)
	})

	// pos == 0 means non-existent item
	item_exists = (item_pos != 0)
	return
}

// IterableMap storage wrapper
type IterableMap struct {
	IterableMapReader
	storage *StorageWrapper
}

// Inits iterable map with prefix, so multiple iterbale maps can coexists thanks to different prefixes
func (self *IterableMap) Init(stor *StorageWrapper, prefix []byte) {
	self.storage = stor
	self.IterableMapReader.Init(&stor.StorageReaderWrapper, prefix)
}

// Creates item from iterable map, return number of items in the iterbale map
func (self *IterableMap) CreateItem(item []byte) uint32 {
	if item_exists, _ := self.itemExists(item); item_exists {
		panic("Item " + string(item) + " already exists")
	}

	// Gets keys array length
	items_count := self.GetCount()

	// items positions are shifetd + 1, item 0 is saved on pos 1, etc... pos 0 is reserved for non-existent item
	new_item_pos := items_count + 1

	// Saves new item into the items array with key -> self.items_storage_prefix + pos
	items_k := Stor_k_1(self.items_storage_prefix, uint32ToBytes(new_item_pos))
	self.storage.Put(items_k, item)
	log.Println("items_k: ", items_k.String(), " -> ", item)

	// Save position of ney item in items array into the items pos mapping
	items_pos_k := Stor_k_1(self.items_pos_storage_prefix, item[:])
	self.storage.Put(items_pos_k, uint32ToBytes(new_item_pos))
	log.Println("items_pos_k: ", items_pos_k.String(), " -> ", uint32ToBytes(new_item_pos))

	// Saves new items count
	self.storage.Put(self.items_count_storage_key, uint32ToBytes(new_item_pos))
	log.Println("items_count_storage_key: ", self.items_count_storage_key.String(), " -> ", uint32ToBytes(new_item_pos))

	return new_item_pos
}

// Removes item from iterable map, returns number of left items in the iterbale map
func (self *IterableMap) RemoveItem(item []byte) uint32 {
	// Gets items count
	items_count := self.GetCount()

	// There are no items saved in storage
	if items_count == 0 {
		panic("Unable to delete item " + string(item) + ". No items in iterable map")
	}

	// Checks if item to be deleted exists
	item_exists, delete_item_pos := self.itemExists(item)
	if item_exists == false {
		panic("Item " + string(item) + " does not exist")
	}
	delete_item_at_pos_k := Stor_k_1(self.items_storage_prefix, uint32ToBytes(delete_item_pos))
	delete_item_pos_k := Stor_k_1(self.items_pos_storage_prefix, item[:])

	// item to be deleted is saved on the last position
	if delete_item_pos == items_count {
		self.storage.Put(delete_item_at_pos_k, nil)
		self.storage.Put(delete_item_pos_k, nil)
		self.storage.Put(self.items_count_storage_key, uint32ToBytes(items_count-1))

		return items_count - 1
	}

	// There is more items saved and item to be deleted is somewhere in the middle

	// Positions are shifted +1 because pos == 0 is reserved to indicate non-existent element
	last_item_at_pos_k := Stor_k_1(self.items_storage_prefix, uint32ToBytes(items_count))

	var last_item []byte
	self.storage.Get(last_item_at_pos_k, func(bytes []byte) {
		last_item = bytes
	})
	if len(last_item) == 0 {
		// This should never happen
		panic("Unable to delete item " + string(item) + ". item not found")
	}

	last_item_pos_k := Stor_k_1(self.items_pos_storage_prefix, last_item[:])

	// Swap item to be deleted with the last item
	self.storage.Put(last_item_pos_k, uint32ToBytes(delete_item_pos))
	self.storage.Put(delete_item_at_pos_k, last_item)

	self.storage.Put(delete_item_pos_k, nil)
	self.storage.Put(last_item_at_pos_k, nil)
	self.storage.Put(self.items_count_storage_key, uint32ToBytes(items_count-1))

	return items_count - 1
}

func uint32ToBytes(val uint32) []byte {
	r := make([]byte, 4)
	for i := uint32(0); i < 4; i++ {
		r[i] = byte((val >> (8 * i)) & 0xff)
	}
	return r
}

func bytesToUint32(val []byte) uint32 {
	r := uint32(0)
	for i := uint32(0); i < 4; i++ {
		r |= uint32(val[i]) << (8 * i)
	}
	return r
}
