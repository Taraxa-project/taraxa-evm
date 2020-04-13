package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

type AccountTrieSchema struct{}

func (AccountTrieSchema) ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte) {
	return enc_storage
}

func (AccountTrieSchema) MaxValueSizeToStoreInTrie() int { return 8 }

type AccountTrieInputHistorical struct {
	AccountTrieSchema
	*BlockDB
	addr *common.Address
}

func (self AccountTrieInputHistorical) GetValue(key *common.Hash) []byte {
	return self.db.GetAccountTrieValue(self.blk_num, self.addr, key)
}

func (self AccountTrieInputHistorical) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetAccountTrieNode(node_hash)
}

type AccountTrieIOPending struct {
	AccountTrieSchema
	*PendingBlockDB
	addr *common.Address
}

func (self AccountTrieIOPending) GetValue(key *common.Hash) []byte {
	return self.db.GetAccountTrieValueLatest(self.addr, key)
}

func (self AccountTrieIOPending) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetAccountTrieNode(node_hash)
}

func (self AccountTrieIOPending) PutValue(key *common.Hash, v []byte) {
	self.db.PutAccountTrieValue(self.blk_num, self.addr, key, v)
	self.db.PutAccountTrieValueLatest(self.addr, key, v)
}

func (self AccountTrieIOPending) DeleteValue(key *common.Hash) {
	self.db.PutAccountTrieValue(self.blk_num, self.addr, key, nil)
	self.db.DeleteAccountTrieValueLatest(self.addr, key)
}

func (self AccountTrieIOPending) PutNode(node_hash *common.Hash, node []byte) {
	self.db.PutAccountTrieNode(node_hash, node)
}
