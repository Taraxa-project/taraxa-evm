package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"runtime"
)

// TODO key versioning
// TODO merge with statedb
type Database struct {
	db          ethdb.Database
	last_tr     *trie.Trie
	tasks       chan func()
	batch       ethdb.Batch
	committed   map[string][]byte
	uncommitted map[string][]byte
	mem_only    bool
}

func NewDatabase(db ethdb.Database) *Database {
	self := &Database{
		db:          db,
		tasks:       make(chan func(), 4096*64),
		committed:   make(map[string][]byte),
		uncommitted: make(map[string][]byte),
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

func (self *Database) GetAccTrie() *trie.Trie {
	return self.last_tr
}

func (self *Database) ToggleMemOnly() {
	util.Assert(!self.mem_only)
	self.mem_only = !self.mem_only
}

func (self *Database) OpenStorageTrie(root []byte, owner_addr common.Address) *trie.Trie {
	return trie.NewTrie(root, &acc_storage_trie_schema{owner_addr}, trie_db{self}, 0)
}

func (self *Database) OpenTrie(root []byte) *trie.Trie {
	if self.last_tr == nil {
		self.last_tr = trie.NewTrie(root, main_trie_schema{rlp.NewEncoder(rlp.EncoderConfig{1 << 8, 1})}, trie_db{self}, 0)
	}
	return self.last_tr
}

func (self *Database) PutAsync(k, v []byte) {
	if self.mem_only {
		self.uncommitted[string(k)] = v
		return
	}
	self.tasks <- func() {
		if self.batch == nil {
			self.batch = self.db.NewBatch()
		}
		util.PanicIfNotNil(self.batch.Put(k, v))
	}
}

func (self *Database) GetCommitted(k []byte) []byte {
	if v, ok := self.committed[binary.StringView(k)]; ok {
		return v
	}
	ret, err := self.db.Get(k)
	util.PanicIfNotNil(err)
	return ret
}

func (self *Database) Commit() {
	if self.mem_only {
		for k, v := range self.uncommitted {
			self.committed[k] = v
			delete(self.uncommitted, k)
		}
		return
	}
	ch := make(chan byte)
	self.tasks <- func() {
		defer close(ch)
		if self.batch != nil {
			util.PanicIfNotNil(self.batch.Write())
			self.batch = nil
		}
	}
	<-ch
}

type trie_db struct {
	*Database
}

func (self trie_db) PutAsync(col trie.StorageColumn, key, value []byte) {
	self.Database.PutAsync(binary.Concat([]byte{col}, key...), value)
}

func (self trie_db) DeleteAsync(col trie.StorageColumn, key []byte) {
	self.PutAsync(col, key, nil)
}

func (self trie_db) GetCommitted(col trie.StorageColumn, key []byte) []byte {
	return self.Database.GetCommitted(binary.Concat([]byte{col}, key...))
}

type main_trie_schema struct {
	encoder *rlp.Encoder
}

func (self main_trie_schema) FlatKey(hashed_key []byte) []byte {
	return hashed_key
}

func (self main_trie_schema) StorageToHashEncoding(enc_storage []byte) (enc_hash []byte) {
	return enc_storage_2_hash(self.encoder, enc_storage)
}

func (self main_trie_schema) MaxStorageEncSizeToStoreInTrie() int {
	return 8
}

type acc_storage_trie_schema struct {
	acc_addr common.Address
}

func (self *acc_storage_trie_schema) FlatKey(hashed_key []byte) []byte {
	return binary.Concat(self.acc_addr[:], hashed_key...)
}

func (self *acc_storage_trie_schema) StorageToHashEncoding(enc_storage []byte) (enc_hash []byte) {
	return enc_storage
}

func (self *acc_storage_trie_schema) MaxStorageEncSizeToStoreInTrie() int {
	return 8
}
