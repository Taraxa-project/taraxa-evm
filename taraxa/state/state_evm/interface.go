package state_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"math/big"
)

type AccountChange struct {
	state_common.Account
	Code         []byte
	CodeDirty    bool
	StorageDirty AccountStorage
}

type AccountStorage = map[common.Hash]*big.Int

type Input interface {
	GetCode(code_hash *common.Hash) []byte
	GetAccount(addr *common.Address) (state_common.Account, bool)
	GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int
}

type Output interface {
	OnAccountChanged(addr common.Address, change AccountChange)
	OnAccountDeleted(addr common.Address)
}
