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
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/Taraxa-project/taraxa-evm/trie"
	lru "github.com/hashicorp/golang-lru"
	"runtime"
	"sync"
)

const MaxTrieCacheGen = uint16(1000)
const codeSizeCacheSize = 100000

type Database struct {
	db            ethdb.Database
	codeSizeCache *lru.Cache
	last_tr       *trie.Trie
	last_tr_init  sync.Once
	tasks         chan func()
	batch         ethdb.Batch
	batch_map     map[string]uint16
	uncommitted   sync.Map
}

type versioned_value = struct {
	ver uint16
	val []byte
	sync.RWMutex
}

func NewDatabase(db ethdb.Database) *Database {
	csc, err := lru.New(codeSizeCacheSize)
	util.PanicIfNotNil(err)
	self := &Database{
		db:            db,
		codeSizeCache: csc,
		tasks:         make(chan func(), 4096*64),
	}
	runtime.SetFinalizer(self, func(self *Database) {
		self.tasks <- nil
	})
	go func() {
		for {
			t := <-self.tasks
			if t == nil {
				return
			}
			t()
		}
	}()
	return self
}

func (self *Database) OpenStorageTrie(root *common.Hash, owner_addr *common.Address) *trie.Trie {
	return trie.New(root, trie_db{db: self}, 0, acc_trie_storage_strat(*owner_addr))
}

func (self *Database) OpenTrie(root *common.Hash) *trie.Trie {
	self.last_tr_init.Do(func() {
		self.last_tr = trie.New(root, trie_db{db: self}, MaxTrieCacheGen, main_trie_storage_strat(0))
	})
	return self.last_tr
}

func (self *Database) ContractCode(hash []byte) ([]byte, error) {
	if cached, ok := self.codeSizeCache.Get(binary.StringView(hash)); ok {
		return cached.([]byte), nil
	}
	code, err := self.Get(hash)
	if err == nil {
		self.codeSizeCache.Add(string(hash), code)
	}
	return code, err
}

func (self *Database) CodeSize(hash []byte) (int, error) {
	code, err := self.ContractCode(hash)
	return len(code), err
}

func (self *Database) PutAsync(k, v []byte) {
	k_str := string(k)
	bid := versioned_value{ver: 1, val: v}
	bid.Lock()
	ver := bid.ver
	for {
		actual, loaded := self.uncommitted.LoadOrStore(k_str, &bid)
		if !loaded {
			bid.Unlock()
		} else {
			vv := actual.(*versioned_value)
			vv.Lock()
			if vv.ver == 0 {
				vv.Unlock()
				continue
			}
			vv.val = v
			vv.ver++
			ver = vv.ver
			vv.Unlock()
		}
		break
	}
	self.tasks <- func() {
		if self.batch == nil {
			self.batch = self.db.NewBatch()
			self.batch_map = make(map[string]uint16)
		}
		self.batch_map[k_str] = ver
		util.PanicIfNotNil(self.batch.Put(k, v))
	}
}

func (self *Database) GetCommitted(k []byte) ([]byte, error) {
	return self.db.Get(k)
}

func (self *Database) Get(k []byte) ([]byte, error) {
	if v, ok := self.uncommitted.Load(binary.StringView(k)); ok {
		v := v.(*versioned_value)
		v.RLock()
		defer v.RUnlock()
		return v.val, nil
	}
	return self.GetCommitted(k)
}

func (self *Database) CommitAsync() {
	self.tasks <- func() {
		if self.batch == nil {
			return
		}
		util.PanicIfNotNil(self.batch.Write())
		for k, batched_ver := range self.batch_map {
			v, _ := self.uncommitted.Load(k)
			vv := v.(*versioned_value)
			vv.Lock()
			if vv.ver == batched_ver {
				self.uncommitted.Delete(k)
				vv.ver = 0
			}
			vv.Unlock()
		}
		self.batch = nil
		self.batch_map = nil
	}
}

func (self *Database) Join() {
	ch := make(chan byte)
	self.tasks <- func() {
		close(ch)
	}
	<-ch
}

type trie_db struct {
	db *Database
}

func (self trie_db) Put(key []byte, value []byte) error {
	self.db.PutAsync(key, value)
	return nil
}

func (self trie_db) Get(key []byte) ([]byte, error) {
	return self.db.Get(key)
}

type main_trie_storage_strat byte

func (main_trie_storage_strat) OriginKeyToMPTKey(key []byte) (ret []byte, ret_release func(), err error) {
	ret, ret_release = util.Keccak256Pooled(key)
	return
}

func (main_trie_storage_strat) MPTKeyToFlat(mpt_key []byte) (flat_key []byte, err error) {
	return binary.Concat(binary.BytesView("main_tr_"), mpt_key...), nil
}

type acc_trie_storage_strat common.Address

func (self acc_trie_storage_strat) OriginKeyToMPTKey(key []byte) (ret []byte, ret_release func(), err error) {
	ret, ret_release = util.Keccak256Pooled(key)
	return
}

func (self acc_trie_storage_strat) MPTKeyToFlat(mpt_key []byte) (ret []byte, err error) {
	return binary.Concat(binary.Concat(binary.BytesView("storage_tr_"), self[:]...), mpt_key...), nil
}
