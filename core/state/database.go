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
	return trie.New(root, self, 0, acc_trie_storage_strat(*owner_addr))
}

func (self *Database) OpenTrie(root *common.Hash) *trie.Trie {
	self.last_tr_init.Do(func() {
		self.last_tr = trie.New(root, self, MaxTrieCacheGen, main_trie_storage_strat(0))
	})
	return self.last_tr
}

func (self *Database) ContractCode(hash []byte) ([]byte, error) {
	if cached, ok := self.codeSizeCache.Get(binary.StringView(hash)); ok {
		return cached.([]byte), nil
	}
	code, err := self.GetCommitted(hash)
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
	self.tasks <- func() {
		if self.batch == nil {
			self.batch = self.db.NewBatch()
		}
		util.PanicIfNotNil(self.batch.Put(k, v))
	}
}

func (self *Database) GetCommitted(k []byte) ([]byte, error) {
	return self.db.Get(k)
}

func (self *Database) Commit() {
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
