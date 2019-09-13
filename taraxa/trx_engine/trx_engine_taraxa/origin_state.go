package trx_engine_taraxa

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"math/big"
	"sync"
)

type Cache interface {
	ComputeIfAbsentAndGet(key interface{}, init func() interface{}) interface{}
}
type MakeCache = func() Cache

type OriginState struct {
	opts          *MakeOriginStateOptions
	makeCache     MakeCache
	trieDB        *trie.Database
	accountCache  Cache
	codeCache     Cache
	codeSizeCache Cache
	accountTrie   *trie.SecureTrie
}

type MakeOriginStateOptions = struct {
	AccountTrieCacheLimit uint16
	StorageTrieCacheLimit uint16
}

func MakeOriginState(root common.Hash, db *trie.Database, makeCache MakeCache, opts *MakeOriginStateOptions) (*OriginState, error) {
	if opts == nil {
		opts = &MakeOriginStateOptions{}
	}
	if opts.AccountTrieCacheLimit == 0 {
		opts.AccountTrieCacheLimit = 120
	}
	if opts.StorageTrieCacheLimit == 0 {
		opts.StorageTrieCacheLimit = 120
	}
	accountTrie, err := trie.NewSecure(root, db, opts.AccountTrieCacheLimit)
	if err != nil {
		return nil, err
	}
	return &OriginState{
		opts:         opts,
		makeCache:    makeCache,
		trieDB:       db,
		accountTrie:  accountTrie,
		accountCache: makeCache(),
	}, nil
}

func (copy OriginState) Copy() *OriginState {
	copy.accountTrie = copy.accountTrie.Copy()
	return &copy
}

func (this *OriginState) LoadAccount(addr common.Address) (ret *OriginAccount, err error) {
	val := this.accountCache.ComputeIfAbsentAndGet(addr, func() interface{} {
		var accRlp []byte
		if accRlp, err = this.accountTrie.TryGet(addr[:]); err != nil || len(accRlp) == 0 {
			return nil
		}
		var accTrieObject state.Account
		if err = rlp.DecodeBytes(accRlp, &accTrieObject); err != nil {
			return (*OriginAccount)(nil)
		}
		acc := &OriginAccount{
			state:   this,
			Nonce:   accTrieObject.Nonce,
			Balance: accTrieObject.Balance,
		}
		codeHash := common.BytesToHash(accTrieObject.CodeHash)
		if codeHash != EmptyCodeHash {
			acc.CodeHash = &codeHash
		}
		if accTrieObject.Root != EmptyTrieRoot {
			acc.Storage = &ReadOnlyStorage{
				SharedStorage: &SharedStorage{Root: accTrieObject.Root},
			}
		}
		return acc
	})
	if err == nil {
		ret = val.(*OriginAccount)
	}
	return
}

type OriginAccount struct {
	state    *OriginState
	Nonce    uint64
	Balance  *big.Int
	CodeHash *common.Hash
	Storage  *ReadOnlyStorage
}

func (copy OriginAccount) Copy() *OriginAccount {
	if copy.Storage != nil {
		copy.Storage = copy.Storage.Copy()
	}
	return &copy
}

func (this *OriginAccount) GetCodeHash() common.Hash {
	if !this.HasCode() {
		return EmptyCodeHash
	}
	return *this.CodeHash
}

func (this *OriginAccount) GetCode() (ret []byte, err error) {
	if !this.HasCode() {
		return
	}
	val := this.state.codeCache.ComputeIfAbsentAndGet(*this.CodeHash, func() (ret interface{}) {
		ret, err = this.state.trieDB.Node(*this.CodeHash)
		return
	})
	if err == nil {
		ret = val.([]byte)
	}
	return
}

func (this *OriginAccount) GetCodeSize() (ret int, err error) {
	if !this.HasCode() {
		return
	}
	val := this.state.codeSizeCache.ComputeIfAbsentAndGet(*this.CodeHash, func() (ret interface{}) {
		var code []byte
		code, err = this.state.trieDB.Node(*this.CodeHash)
		return len(code)
	})
	if err == nil {
		ret = val.(int)
	}
	return
}

func (this *OriginAccount) HasCode() bool {
	return this.CodeHash != nil
}

func (this *OriginAccount) GetStorage(key common.Hash) (ret *common.Hash, err error) {
	if !this.HasStorage() {
		return
	}
	return this.Storage.Get(this.state, key)
}

func (this *OriginAccount) HasStorage() bool {
	return this.Storage != nil
}

type ReadOnlyStorage struct {
	LocalTrie     *trie.SecureTrie
	SharedStorage *SharedStorage
}

func (copy ReadOnlyStorage) Copy() *ReadOnlyStorage {
	copy.LocalTrie = nil
	return &copy
}

func (this *ReadOnlyStorage) Get(state *OriginState, key common.Hash) (ret *common.Hash, err error) {
	if this.LocalTrie == nil {
		if this.LocalTrie, err = this.SharedStorage.LoadAndCopyTrie(state); err != nil {
			return
		}
	}
	val := this.SharedStorage.Cache.ComputeIfAbsentAndGet(key, func() interface{} {
		var valueRlp []byte
		if valueRlp, err = this.LocalTrie.TryGet(key[:]); err != nil {
			return nil
		}
		ret := new(common.Hash)
		if len(valueRlp) == 0 {
			return ret
		}
		var valueBytes []byte
		if _, valueBytes, _, err = rlp.Split(valueRlp); err != nil {
			return nil
		}
		ret.SetBytes(valueBytes)
		return ret
	})
	if err != nil {
		return
	}
	return val.(*common.Hash), nil
}

type SharedStorage struct {
	Root  common.Hash
	Init  sync.Once
	Trie  *trie.SecureTrie
	Cache Cache
}

func (this *SharedStorage) LoadAndCopyTrie(state *OriginState) (ret *trie.SecureTrie, err error) {
	this.Init.Do(func() {
		if this.Trie, err = trie.NewSecure(this.Root, state.trieDB, state.opts.StorageTrieCacheLimit); err != nil {
			return
		}
		this.Cache = state.makeCache()
	})
	if err != nil {
		return
	}
	return this.Trie.Copy(), nil
}
