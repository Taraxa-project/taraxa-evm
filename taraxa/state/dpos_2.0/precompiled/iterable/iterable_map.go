package iterable

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/storage"
)

type StorageData = map[*common.Hash][]byte

type Serializable interface {
	// Serialize() []byte
	Serialize(pos *common.Hash, out StorageData)
}

type Map[T Serializable] struct {
	storage          storage.StorageWrapper
	storage_position *common.Hash
	keys             []common.Address
	mapping          map[common.Address]T
}

func (m *Map[T]) Init(pos *common.Hash, s storage.StorageWrapper) {
	m.storage_position = pos
	m.storage = s
}

func (m *Map[T]) Load() {
	m.storage.Get(&common.ZeroHash, func(bytes []byte) {

	})
}

func (m *Map[T]) Save() {
	result := make(StorageData)

	// Save array
	array_key := keccak256.Hash(m.storage_position.Bytes(), big.NewInt(0).Bytes())
	result[array_key] = big.NewInt(int64(len(m.keys))).Bytes()
	for i, v := range m.keys {
		key := keccak256.Hash(big.NewInt(int64(i)).Bytes(), array_key.Bytes())
		result[key] = v.Bytes()
	}

	for i, v := range result {
		m.storage.Put(i, v)
	}
}

func (m *Map[T]) Get(key *common.Address) T {
	return m.mapping[*key]
}

func (m *Map[T]) GetKeys() []common.Address {
	return m.keys
}

func exists[T Serializable](key *common.Address, mapping *map[common.Address]T) bool {
	_, exists := (*mapping)[*key]
	return exists
}

func (m *Map[T]) PushBack(key *common.Address, value *T) bool {
	if !exists(key, &m.mapping) {
		return false
	}
	m.keys = append(m.keys, *key)
	m.mapping[*key] = *value

	return true
}

func (m *Map[T]) PushBackOrModify(key *common.Address, value *T) {
	if !exists(key, &m.mapping) {
		m.keys[len(m.keys)] = *key
	}

	m.mapping[*key] = *value
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

func (m *Map[T]) Erase(key *common.Address) bool {
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
