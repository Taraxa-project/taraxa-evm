package trx_engine_taraxa

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
)

type AccountField byte

const (
	balance AccountField = iota
	nonce
	storage
	code
	AccountField_count
)

type AccountFieldSet = [AccountField_count]bool
type AccountSet = map[common.Address]bool
type Code = struct {
	Hash  common.Hash
	Value []byte
	Size  int
}
type StorageCell = struct {
	OriginValue *common.Hash
	Value       *common.Hash
}
type StorageKeySet = map[common.Hash]bool
type Storage = map[common.Hash]*StorageCell
type Preimages = map[common.Hash][]byte
type Logs = []*types.Log



var EmptyHash = crypto.Keccak256Hash(nil)
var EmptyCodeHash = EmptyHash
var EmptyTrieRoot = EmptyHash
