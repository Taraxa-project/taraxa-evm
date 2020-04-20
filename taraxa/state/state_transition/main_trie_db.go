package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type main_trie_db struct {
	state_common.MainTrieSchema
	*StateTransition
}

func (self main_trie_db) GetValue(key *common.Hash) []byte {
	return self.db.GetMainTrieValueLatest(key)
}

func (self main_trie_db) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetMainTrieNode(node_hash)
}

func (self main_trie_db) PutValue(key *common.Hash, v []byte) {
	self.db.PutMainTrieValue(self.curr_blk_num, key, v)
	self.db.PutMainTrieValueLatest(key, v)
}

func (self main_trie_db) DeleteValue(key *common.Hash) {
	self.db.PutMainTrieValue(self.curr_blk_num, key, nil)
	self.db.DeleteMainTrieValueLatest(key)
}

func (self main_trie_db) PutNode(node_hash *common.Hash, node []byte) {
	self.db.PutMainTrieNode(node_hash, node)
}
