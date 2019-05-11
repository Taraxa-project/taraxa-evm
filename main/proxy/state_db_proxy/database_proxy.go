package state_db_proxy

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
)

type DatabaseProxy struct {
	state.Database
	Decorators     *proxy.Decorators
	TrieDecorators *proxy.Decorators
}

func (this DatabaseProxy) OpenTrie(root common.Hash) (t state.Trie, e error) {
	after := this.Decorators.BeforeCall("OpenTrie", &root)
	defer after(&t, &e)
	trie, err := this.Database.OpenTrie(root)
	return TrieProxy{trie, this.TrieDecorators}, err
}

func (this DatabaseProxy) OpenStorageTrie(addrHash, root common.Hash) (t state.Trie, e error) {
	after := this.Decorators.BeforeCall("OpenStorageTrie", &addrHash, &root)
	defer after(&t, &e)
	trie, err := this.Database.OpenStorageTrie(addrHash, root)
	return TrieProxy{trie, this.TrieDecorators}, err
}

func (this DatabaseProxy) CopyTrie(trie state.Trie) state.Trie {
	return TrieProxy{this.Database.CopyTrie(trie), this.TrieDecorators}
}

func (this DatabaseProxy) ContractCode(addrHash, codeHash common.Hash) (b []byte, e error) {
	after := this.Decorators.BeforeCall("ContractCode", &addrHash, &codeHash)
	defer after(&b, &e)
	return this.Database.ContractCode(addrHash, codeHash)
}

type TrieProxy struct {
	state.Trie
	Decorators *proxy.Decorators
}

func (this TrieProxy) TryGet(key []byte) (b []byte, e error) {
	after := this.Decorators.BeforeCall("TryGet", &key)
	defer after(&b, &e)
	return this.Trie.TryGet(key)
}
