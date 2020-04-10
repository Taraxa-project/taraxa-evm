package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

type AccountTrieSchema struct{}

func (AccountTrieSchema) ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte) {
	return enc_storage
}

func (AccountTrieSchema) MaxValueSizeToStoreInTrie() int { return 8 }

type AccountTrieInput struct {
	BlockState
	addr *common.Address
}

func (self *AccountTrieInput) GetValue(key *common.Hash) []byte {
	return self.db.GetAccountTrieValue(self.blk_num, self.addr, key)
}

func (self *AccountTrieInput) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetAccountTrieNode(node_hash)
}

type AccountTrieOutput struct {
	BlockState
	addr *common.Address
}

func (self *AccountTrieOutput) PutValue(key *common.Hash, v []byte) {
	self.db.PutAccountTrieValue(self.blk_num, self.addr, key, v)
}

func (self *AccountTrieOutput) DeleteValue(key *common.Hash) {
	self.PutValue(key, nil)
}

func (self *AccountTrieOutput) PutNode(node_hash *common.Hash, node []byte) {
	self.db.PutAccountTrieNode(node_hash, node)
}
