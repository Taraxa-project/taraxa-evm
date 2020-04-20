package state_historical

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type main_trie_db struct {
	state_common.MainTrieSchema
	BlockDB
}

func (self main_trie_db) GetValue(key *common.Hash) []byte {
	return self.db.GetMainTrieValue(self.blk_num, key)
}

func (self main_trie_db) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetMainTrieNode(node_hash)
}
