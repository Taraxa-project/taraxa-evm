// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/Taraxa-project/taraxa-evm/trie"
	lru "github.com/hashicorp/golang-lru"
)

const MaxTrieCacheGen = uint16(500)
const codeSizeCacheSize = 100000

func NewDatabase(db ethdb.Database) *Database {
	csc, err := lru.New(codeSizeCacheSize)
	util.PanicIfNotNil(err)
	return &Database{
		db:            db,
		codeSizeCache: csc,
	}
}

type Database struct {
	db            ethdb.Database
	codeSizeCache *lru.Cache
	batch         ethdb.Batch
}

func (self *Database) OpenStorageTrie(root *common.Hash, owner_addr *common.Address) (*trie.Trie, error) {
	return trie.New(root, trie_db{self, make(map[string][]byte)}, 0, acc_trie_storage_strat(*owner_addr))
}

func (self *Database) OpenTrie(root *common.Hash) (*trie.Trie, error) {
	return trie.New(root, trie_db{self, make(map[string][]byte)}, MaxTrieCacheGen, main_trie_storage_strat(0))
}

func (self *Database) ContractCode(hash []byte) ([]byte, error) {
	code, err := self.db.Get(hash)
	if err == nil {
		self.codeSizeCache.Add(string(hash), len(code))
	}
	return code, err
}

func (self *Database) CodeSize(hash []byte) (int, error) {
	if cached, ok := self.codeSizeCache.Get(binary.StringView(hash)); ok {
		return cached.(int), nil
	}
	code, err := self.ContractCode(hash)
	return len(code), err
}

func (self *Database) PutCode(hash, code []byte) error {
	return self.put(hash, code)
}

func (self *Database) Commit() error {
	if self.batch == nil {
		return nil
	}
	err := self.batch.Write()
	self.batch = nil
	return err
}

func (self *Database) put(k, v []byte) error {
	if self.batch == nil {
		self.batch = self.db.NewBatch()
	}
	return self.batch.Put(k, v)
}

type trie_db struct {
	*Database
	cache map[string][]byte
}

func (self trie_db) Put(key []byte, value []byte) error {
	self.cache[string(key)] = value
	return self.put(key, value)
}

func (self trie_db) Get(key []byte) ([]byte, error) {
	if v, ok := self.cache[binary.StringView(key)]; ok {
		return v, nil
	}
	return self.db.Get(key)
}

func (self trie_db) GetCommitted(key []byte) ([]byte, error) {
	return self.db.Get(key)
}

type main_trie_storage_strat byte

func (main_trie_storage_strat) OriginKeyToMPTKey(key []byte) (mpt_key []byte, err error) {
	return crypto.Keccak256(key), nil
}

func (main_trie_storage_strat) MPTKeyToFlat(mpt_key []byte) (flat_key []byte, err error) {
	return binary.Concat(binary.BytesView("main_tr_"), mpt_key...), nil
}

func (main_trie_storage_strat) UseFlat() bool {
	return true
}

type acc_trie_storage_strat common.Address

func (self acc_trie_storage_strat) OriginKeyToMPTKey(key []byte) (mpt_key []byte, err error) {
	return crypto.Keccak256(key), nil
}

func (self acc_trie_storage_strat) MPTKeyToFlat(mpt_key []byte) (flat_key []byte, err error) {
	return binary.Concat(binary.Concat(binary.BytesView("storage_tr_"), self[:]...), mpt_key...), nil
}

func (acc_trie_storage_strat) UseFlat() bool {
	return true
}
