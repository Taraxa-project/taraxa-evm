package state_evm

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_trie"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/common"
)

type AccountChange struct {
	state_trie.Account
	Code            state_common.ManagedSlice
	CodeDirty       bool
	StorageDirty    EVMStorage
	RawStorageDirty RawStorage
}
type EVMStorage = map[bigutil.UnsignedStr]*big.Int
type RawStorage = map[common.Hash][]byte

type Input interface {
	GetCode(*common.Hash) state_common.ManagedSlice
	GetRawAccount(*common.Address, func([]byte))
	GetAccountStorage(*common.Address, *common.Hash, func([]byte))
}
type Sink interface {
	StartMutation(*common.Address) AccountMutation
	Delete(*common.Address)
}
type AccountMutation interface {
	Update(AccountChange)
}
type AccountMutations interface {
	ForEachMutationWithDuplicates(func(AccountMutation))
}
