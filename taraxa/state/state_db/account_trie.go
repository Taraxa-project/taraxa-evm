package state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type AccountTrieSchema struct{}

func (AccountTrieSchema) ValueStorageToHashEncoding(enc_storage []byte) []byte {
	return rlp.ToRLPStringSimple(enc_storage)
}

func (AccountTrieSchema) MaxValueSizeToStoreInTrie() int { return 8 }

type acc_storage_trie_value struct{ enc_storage, enc_hash []byte }

func NewAccStorageTrieValue(enc_storage []byte) (ret acc_storage_trie_value) {
	ret.enc_storage, ret.enc_hash = enc_storage, rlp.ToRLPStringSimple(enc_storage)
	return
}

func (self acc_storage_trie_value) EncodeForTrie() (enc_storage, enc_hash []byte) {
	enc_storage, enc_hash = self.enc_storage, self.enc_hash
	return
}

type AccountTrieInputAdapter struct {
	Addr *common.Address
	Reader
}

func (self AccountTrieInputAdapter) GetValue(key *common.Hash, cb func(v []byte)) {
	self.Get(COL_acc_trie_value, acc_trie_db_key(self.Addr, key), cb)
}

func (self AccountTrieInputAdapter) GetNode(node_hash *common.Hash, cb func([]byte)) {
	self.Get(COL_acc_trie_node, node_hash, cb)
}

type AccountTrieIOAdapter struct {
	Addr *common.Address
	ReadWriter
}

func (self AccountTrieIOAdapter) GetValue(key *common.Hash, cb func(v []byte)) {
	self.Get(COL_acc_trie_value, acc_trie_db_key(self.Addr, key), cb)
}

func (self AccountTrieIOAdapter) GetNode(node_hash *common.Hash, cb func([]byte)) {
	self.Get(COL_acc_trie_node, node_hash, cb)
}

func (self AccountTrieIOAdapter) PutValue(key *common.Hash, v []byte) {
	self.Put(COL_acc_trie_value, acc_trie_db_key(self.Addr, key), v)
}

func (self AccountTrieIOAdapter) PutNode(node_hash *common.Hash, node []byte) {
	self.Put(COL_acc_trie_node, node_hash, node)
}

func acc_trie_db_key(addr *common.Address, key *common.Hash) *common.Hash {
	return keccak256.Hash(addr[:], key[:])
}
