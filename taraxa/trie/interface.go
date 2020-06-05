package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Schema interface {
	ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxValueSizeToStoreInTrie() int
}

type ReadOnlyDB interface {
	Schema
	GetValue(key *common.Hash) []byte
	GetNode(node_hash *common.Hash) []byte
}

type DB interface {
	ReadOnlyDB
	PutValue(key *common.Hash, v []byte)
	DeleteValue(key *common.Hash)
	PutNode(node_hash *common.Hash, node []byte)
}
