package state_db_rocksdb

import (
	"encoding/binary"
	"unsafe"

	"github.com/linxGnu/grocksdb"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

type VersionedKey [common.HashLength + unsafe.Sizeof(types.BlockNum(0))]byte

func (self *VersionedKey) SetKey(prefix *common.Hash) {
	copy(self[:], prefix[:])
}

func (self *VersionedKey) SetVersion(block_num types.BlockNum) {
	binary.BigEndian.PutUint64(self[common.HashLength:], block_num)
}

type VersionedReadContext struct {
	itr        *grocksdb.Iterator
	key_buffer VersionedKey
}

func (self *VersionedReadContext) Close() {
	self.itr.Close()
}
