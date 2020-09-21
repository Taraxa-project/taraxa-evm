package state_transition

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/core/vm"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type dpos_storage_adapter struct{ *StateTransition }

func (self dpos_storage_adapter) SubBalance(address *common.Address, b state_common.TaraxaBalance) bool {
	acc := self.evm_state.GetAccountConcrete(address)
	b_big := new(big.Int).SetUint64(b)
	if vm.BalanceGTE(acc, b_big) {
		acc.SubBalance(b_big)
		return true
	}
	return false
}

func (self dpos_storage_adapter) Put(address *common.Address, k *common.Hash, v []byte) {
	self.evm_state.GetAccountConcrete(address).SetStateRawIrreversibly(k, v)
}

func (self dpos_storage_adapter) Get(address *common.Address, k *common.Hash, cb func([]byte)) {
	self.last_block_reader.GetAccountStorage(address, k, cb)
}

func (self dpos_storage_adapter) ForEach(addr *common.Address, cb func(*common.Hash, []byte)) {
	self.last_block_reader.ForEachStorage(addr, cb)
}
