package state_db_proxy

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy"
)

type DatabaseProxy struct {
	state.Database
	*proxy.BaseProxy
	TrieProxy *proxy.BaseProxy
}

func (this DatabaseProxy) OpenTrie(root common.Hash) (t state.Trie, e error) {
	defer this.CallDecorator("OpenTrie", &root)(&t, &e)
	trie, err := this.Database.OpenTrie(root)
	return TrieProxy{trie, this.TrieProxy}, err
}

func (this DatabaseProxy) OpenStorageTrie(addrHash, root common.Hash) (t state.Trie, e error) {
	defer this.CallDecorator("OpenStorageTrie", &addrHash, &root)(&t, &e)
	trie, err := this.Database.OpenStorageTrie(addrHash, root)
	return TrieProxy{trie, this.TrieProxy}, err
}

func (this DatabaseProxy) CopyTrie(trie state.Trie) state.Trie {
	return TrieProxy{this.Database.CopyTrie(trie), this.TrieProxy}
}

func (this DatabaseProxy) ContractCode(addrHash, codeHash common.Hash) (b []byte, e error) {
	defer this.CallDecorator("ContractCode", &addrHash, &codeHash)(&b, &e)
	return this.Database.ContractCode(addrHash, codeHash)
}

type TrieProxy struct {
	state.Trie
	*proxy.BaseProxy
}

func (this TrieProxy) TryGet(key []byte) (b []byte, e error) {
	defer this.CallDecorator("TryGet", &key)(&b, &e)
	return this.Trie.TryGet(key)
}
