package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Schema interface {
	ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxValueSizeToStoreInTrie() int
}
type ReadTxn interface {
	GetValue(*common.Hash, func([]byte))
	GetNode(*common.Hash, func([]byte))
}
type WriteTxn interface {
	ReadTxn
	PutValue(*common.Hash, []byte)
	PutNode(*common.Hash, []byte)
}
