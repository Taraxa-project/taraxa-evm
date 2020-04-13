package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Schema interface {
	ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxValueSizeToStoreInTrie() int
}

type Reader interface {
	GetValue(key *common.Hash) []byte
	GetNode(node_hash *common.Hash) []byte
}

type Writer interface {
	PutValue(key *common.Hash, v []byte)
	DeleteValue(key *common.Hash)
	PutNode(node_hash *common.Hash, node []byte)
}

type ReadOnlyDB interface {
	Schema
	Reader
}

type DB interface {
	ReadOnlyDB
	Writer
}
