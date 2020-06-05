package state_historical

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type DB struct{ state_common.DB }

func (self DB) AtBlock(blk_num types.BlockNum) BlockDB {
	return BlockDB{self.DB, blk_num}
}
