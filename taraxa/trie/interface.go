package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Input interface {
	GetValue(key *common.Hash) []byte
	GetNode(node_hash *common.Hash) []byte
}

type Output interface {
	PutValue(key *common.Hash, v []byte)
	DeleteValue(key *common.Hash)
	PutNode(node_hash *common.Hash, node []byte)
}

type Schema interface {
	ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxValueSizeToStoreInTrie() int
}
