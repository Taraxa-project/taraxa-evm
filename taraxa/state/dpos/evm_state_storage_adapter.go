package dpos

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"

	"github.com/Taraxa-project/taraxa-evm/common"
)

type EVMStateStorage struct{ *state_evm.EVMState }

func (self EVMStateStorage) SubBalance(address *common.Address, b *big.Int) bool {
	if acc := self.GetAccountConcrete(address); vm.BalanceGTE(acc, b) {
		acc.SubBalance(b)
		return true
	}
	return false
}

func (self EVMStateStorage) AddBalance(address *common.Address, b *big.Int) {
	self.GetAccountConcrete(address).AddBalance(b)
}

func (self EVMStateStorage) Put(address *common.Address, k *common.Hash, v []byte) {
	self.GetAccountConcrete(address).SetStateRawIrreversibly(k, v)
}

func (self EVMStateStorage) Get(address *common.Address, k *common.Hash, cb func([]byte)) {
	self.GetAccountStorageFromDB(address, k, cb)
}

func (self EVMStateStorage) IncrementNonce(address *common.Address) {
	self.GetAccountConcrete(address).IncrementNonce()
}
