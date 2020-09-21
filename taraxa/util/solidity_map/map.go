package solidity_map

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type Storage struct {
	Put func(*common.Hash, []byte)
	Get func(*common.Hash, func([]byte))
}

type Map struct {
	storage Storage
}

func (self *Map) Init(storage Storage) *Map {
	self.storage = storage
	return self
}

func (self *Map) Put(v []byte, k ...[]byte) {
	self.storage.Put(keccak256.Hash(k...), v)
}

func (self *Map) Get(cb func([]byte), k ...[]byte) {
	self.storage.Get(keccak256.Hash(k...), cb)
}
