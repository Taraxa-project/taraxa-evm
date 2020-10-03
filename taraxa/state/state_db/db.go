package state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

type Column = byte

const (
	COL_code Column = iota
	COL_main_trie_node
	COL_main_trie_value
	COL_acc_trie_node
	COL_acc_trie_value
	COL_COUNT
)

type DB interface {
	ReadBlock(types.BlockNum) ReadTx
	WriteBlock(types.BlockNum) WriteTx
}
type ReadTx interface {
	Get(Column, *common.Hash, func([]byte))
	NotifyDoneReading()
}
type WriteTx interface {
	ReadTx
	Put(Column, *common.Hash, []byte)
	Commit() error
}
