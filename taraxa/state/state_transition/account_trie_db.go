package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type account_trie_db struct {
	state_common.AccountTrieSchema
	*StateTransition
	addr *common.Address
}

func (self account_trie_db) GetValue(key *common.Hash) []byte {
	return self.db.GetAccountTrieValueLatest(self.addr, key)
}

func (self account_trie_db) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetAccountTrieNode(node_hash)
}

func (self account_trie_db) PutValue(key *common.Hash, v []byte) {
	self.db.PutAccountTrieValue(self.curr_blk_num, self.addr, key, v)
	self.db.PutAccountTrieValueLatest(self.addr, key, v)
}

func (self account_trie_db) DeleteValue(key *common.Hash) {
	self.db.PutAccountTrieValue(self.curr_blk_num, self.addr, key, nil)
	self.db.DeleteAccountTrieValueLatest(self.addr, key)
}

func (self account_trie_db) PutNode(node_hash *common.Hash, node []byte) {
	self.db.PutAccountTrieNode(node_hash, node)
}
