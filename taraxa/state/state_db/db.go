package state_db

import (
	"fmt"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
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

// TODO a wrapper with common functionality. Delegate only the most low-level stuff to these interfaces
type DB interface {
	GetBlockState(types.BlockNum) Reader
	GetLatestState() LatestState
}

type ErrFutureBlock util.ErrorString

func GetBlockState(db DB, blk_n types.BlockNum) ExtendedReader {
	last_committed_blk_n := db.GetLatestState().GetCommittedDescriptor().BlockNum
	if last_committed_blk_n < blk_n {
		panic(ErrFutureBlock(fmt.Sprint("Requested blk num:", blk_n, ", last committed:", last_committed_blk_n)))
	}
	return ExtendedReader{db.GetBlockState(blk_n)}
}

type LatestState interface {
	GetCommittedDescriptor() StateDescriptor
	BeginPendingBlock() PendingBlockState
	Commit(state_root common.Hash) error
}
type Reader interface {
	Get(Column, *common.Hash, func([]byte))
}
type ReadWriter interface {
	Reader
	Put(Column, *common.Hash, []byte)
}
type PendingBlockState interface {
	ReadWriter
	GetNumber() types.BlockNum
}

type StateDescriptor struct {
	BlockNum  types.BlockNum
	StateRoot common.Hash
}
