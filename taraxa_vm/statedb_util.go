package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
)

type FlushOpts struct {
	deleteEmptyObjects bool
	report             bool
}

func Flush(statedb *state.StateDB, flushOpts func(*FlushOpts)) (stateRoot common.Hash, err error) {
	opts := FlushOpts{
		deleteEmptyObjects: false,
		report:             true,
	}
	if flushOpts != nil {
		flushOpts(&opts)
	}
	if stateRoot, err = statedb.Commit(opts.deleteEmptyObjects); err != nil {
		return
	}
	err = statedb.Database().TrieDB().Commit(stateRoot, opts.report)
	return
}
