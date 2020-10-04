package state_evm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/common"
)

type AccountChange struct {
	state_db.Account
	StorageDirty    EVMStorage
	RawStorageDirty RawStorage
	Code            []byte
	CodeDirty       bool
}
type EVMStorage = map[bigutil.UnsignedStr]*big.Int
type RawStorage = map[common.Hash][]byte

type Input interface {
	GetCode(*common.Hash) []byte
	GetRawAccount(*common.Address, func([]byte))
	GetAccountStorage(*common.Address, *common.Hash, func([]byte))
}
type DBWriter interface {
	StartMutation(*common.Address) AccountMutation
	Delete(*common.Address)
}
type AccountMutation interface {
	Update(AccountChange)
	Commit()
}
