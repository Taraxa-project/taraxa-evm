package state_historical

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type account_trie_db struct {
	state_common.AccountTrieSchema
	BlockDB
	addr *common.Address
}

func (self account_trie_db) GetValue(key *common.Hash) []byte {
	return self.db.GetAccountTrieValue(self.blk_num, self.addr, key)
}

func (self account_trie_db) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetAccountTrieNode(node_hash)
}
