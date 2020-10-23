package state_db_rocksdb

import (
	"encoding/binary"
	"runtime"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/tecbot/gorocksdb"
)

type TrieValueKey [common.HashLength + unsafe.Sizeof(types.BlockNum(0))]byte

func (self *TrieValueKey) SetKey(prefix *common.Hash) {
	copy(self[:], prefix[:])
}

func (self *TrieValueKey) SetBlockNum(block_num types.BlockNum) {
	binary.BigEndian.PutUint64(self[common.HashLength:], block_num)
}

var opts_r_itr = func() *gorocksdb.ReadOptions {
	ret := gorocksdb.NewDefaultReadOptions()
	ret.SetVerifyChecksums(false)
	ret.SetPrefixSameAsStart(true)
	ret.SetFillCache(false)
	return ret
}()
var opts_r = func() *gorocksdb.ReadOptions {
	ret := gorocksdb.NewDefaultReadOptions()
	ret.SetVerifyChecksums(false)
	return ret
}()
var opts_w = gorocksdb.NewDefaultWriteOptions()
var db_opts = func() *gorocksdb.Options {
	ret := gorocksdb.NewDefaultOptions()
	ret.SetErrorIfExists(false)
	ret.SetCreateIfMissing(true)
	ret.SetCreateIfMissingColumnFamilies(true)
	ret.IncreaseParallelism(runtime.NumCPU())
	ret.SetMaxFileOpeningThreads(runtime.NumCPU())
	return ret
}()
var cf_opts_default = gorocksdb.NewDefaultOptions()
