package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Schema interface {
	ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxValueSizeToStoreInTrie() int
}

type ReadOnlyDB interface {
	Schema
	GetValue(*common.Hash, func([]byte))
	GetNode(*common.Hash, func([]byte))
}

type DB interface {
	ReadOnlyDB
	PutValue(*common.Hash, []byte)
	PutNode(*common.Hash, []byte)
}
