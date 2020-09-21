package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

type DB interface {
	NewBlockReadTransaction(types.BlockNum) BlockReadTransaction
	NewBlockCreationTransaction(types.BlockNum) BlockCreationTransaction
}
type BlockReadTransaction interface {
	GetCode(*common.Hash) ManagedSlice
	GetMainTrieNode(*common.Hash, func([]byte))
	GetAccountTrieNode(*common.Hash, func([]byte))
	GetMainTrieValue(*common.Hash, func([]byte))
	GetAccountTrieValue(*common.Hash, func([]byte))
	NotifyDoneReading()
}
type BlockCreationTransaction interface {
	BlockReadTransaction
	PutCode(*common.Hash, []byte)
	PutMainTrieNode(*common.Hash, []byte)
	PutMainTrieValue(*common.Hash, []byte)
	PutAccountTrieNode(*common.Hash, []byte)
	PutAccountTrieValue(*common.Hash, []byte)
}

type ManagedSlice interface {
	Value() []byte
	Free()
}

type SimpleManagedSlice []byte

func (self SimpleManagedSlice) Value() []byte { return self }
func (self SimpleManagedSlice) Free()         {}
