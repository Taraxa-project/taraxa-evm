package state_historical

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type DB struct{ state_common.DB }

func (self *DB) ReadBlock(n types.BlockNum) (*BlockReader, state_common.BlockReadTransaction) {
	tx := self.NewBlockReadTransaction(n)
	return new(BlockReader).SetTransaction(tx), tx
}
