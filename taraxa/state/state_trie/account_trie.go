package state_trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
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

type AccountTrieReadDB struct {
	AccountTrieSchema
	addr  *common.Address
	db_tx state_common.BlockReadTransaction
}

func (self *AccountTrieReadDB) Init(addr *common.Address) *AccountTrieReadDB {
	self.addr = addr
	return self
}

func (self *AccountTrieReadDB) SetTransaction(db_tx state_common.BlockReadTransaction) *AccountTrieReadDB {
	self.db_tx = db_tx
	return self
}

func (self *AccountTrieReadDB) GetValue(key *common.Hash, cb func(v []byte)) {
	self.db_tx.GetAccountTrieValue(keccak256.Hash(self.addr[:], key[:]), cb)
}

func (self *AccountTrieReadDB) GetNode(node_hash *common.Hash, cb func([]byte)) {
	self.db_tx.GetAccountTrieNode(node_hash, cb)
}

type AccountTrieDB struct {
	AccountTrieReadDB
	db_tx state_common.BlockCreationTransaction
}

func (self *AccountTrieDB) Init(addr *common.Address) *AccountTrieDB {
	self.AccountTrieReadDB.Init(addr)
	return self
}

func (self *AccountTrieDB) SetTransaction(db_tx state_common.BlockCreationTransaction) *AccountTrieDB {
	self.AccountTrieReadDB.SetTransaction(db_tx)
	self.db_tx = db_tx
	return self
}

func (self *AccountTrieDB) PutValue(key *common.Hash, v []byte) {
	self.db_tx.PutAccountTrieValue(keccak256.Hash(self.addr[:], key[:]), v)
}

func (self *AccountTrieDB) PutNode(node_hash *common.Hash, node []byte) {
	self.db_tx.PutAccountTrieNode(node_hash, node)
}
