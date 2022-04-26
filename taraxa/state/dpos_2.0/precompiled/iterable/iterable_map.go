package iterable

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/storage_accessor"
)

type StorageData = map[*common.Hash][]byte

func SerializeArray[T StorageSerializable](pos *common.Hash, arr []T, out StorageData) {
	accessor := new(storage_accessor.StorageAccessor)
	accessor.SetPos(pos)

	size_pos := accessor.Array().ArraySize()
	// write size
	out[&size_pos] = big.NewInt(int64(len(arr))).Bytes()
	//write elements
	for i, v := range arr {
		key := accessor.Array().At(i).Key()
		v.Serialize(&key, out)
	}
}

type StorageSerializable interface {
	// Serialize() []byte
	Load(pos *common.Hash, get func(*common.Hash, func([]byte)))
	Serialize(pos *common.Hash, out StorageData)
}

type AddressMap[T StorageSerializable] struct {
	reader           storage.StorageReaderWrapper
	storage_position *common.Hash
	keys             []common.Address
	mapping          map[common.Address]*T
}

func (m *AddressMap[T]) Init(pos *common.Hash, r storage.StorageReaderWrapper) {
	m.storage_position = pos
	m.reader = r
	m.Load(pos, m.reader.Get)
}

func (m *AddressMap[T]) Load(pos *common.Hash, get func(*common.Hash, func([]byte))) {
	// m.storage.Get(&common.ZeroHash, func(bytes []byte) {

	// })
}

// func (m *AddressMap[T]) Save() {
// 	result := make(StorageData)

// 	// Save array
// 	array_key := keccak256.Hash(m.storage_position.Bytes(), big.NewInt(0).Bytes())
// 	result[array_key] = big.NewInt(int64(len(m.keys))).Bytes()
// 	for i, v := range m.keys {
// 		key := keccak256.Hash(big.NewInt(int64(i)).Bytes(), array_key.Bytes())
// 		result[key] = v.Bytes()
// 	}

// 	// for i, v := range result {
// 	// 	m.storage.Put(i, v)
// 	// }
// }

func (m *AddressMap[T]) Serialize(pos *common.Hash, out StorageData) {

}

func (m *AddressMap[T]) Get(key *common.Address) *T {
	return m.mapping[*key]
}

func (m *AddressMap[T]) GetKeys() []common.Address {
	return m.keys
}

func exists[T StorageSerializable](key *common.Address, mapping *map[common.Address]*T) bool {
	_, exists := (*mapping)[*key]
	return exists
}

func (m *AddressMap[T]) PushBack(key *common.Address, value *T) bool {
	if !exists(key, &m.mapping) {
		return false
	}
	m.keys = append(m.keys, *key)
	m.mapping[*key] = value

	return true
}

func (m *AddressMap[T]) PushBackOrModify(key *common.Address, value *T) {
	if !exists(key, &m.mapping) {
		m.keys[len(m.keys)] = *key
	}

	m.mapping[*key] = value
}

func findIndex(s []common.Address, key *common.Address) (bool, int) {
	for i, v := range s {
		if v == *key {
			return true, i
		}
	}
	return false, 0
}

func removeFromSlice(s []common.Address, i int) []common.Address {
	copy(s[i:], s[i+1:])             // Shift s[i+1:] left one index.
	s[len(s)-1] = common.ZeroAddress // Erase last element (write zero value).
	s = s[:len(s)-1]                 // Truncate slice.
	return s
}

func (m *AddressMap[T]) Erase(key *common.Address) bool {
	if !exists(key, &m.mapping) {
		return false
	}
	e, i := findIndex(m.keys, key)
	if !e {
		return false
	}

	m.keys = removeFromSlice(m.keys, i)
	delete(m.mapping, *key)

	return true
}
