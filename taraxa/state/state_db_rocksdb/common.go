package state_db_rocksdb

import (
	"encoding/binary"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

type TrieValueKey [common.HashLength + unsafe.Sizeof(types.BlockNum(0))]byte

func (self *TrieValueKey) SetKey(prefix *common.Hash) {
	copy(self[:], prefix[:])
}

func (self *TrieValueKey) SetBlockNum(block_num types.BlockNum) {
	binary.BigEndian.PutUint64(self[common.HashLength:], block_num)
}
