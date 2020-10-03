package state_transition

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/core/vm"

	"github.com/Taraxa-project/taraxa-evm/common"
)

type dpos_storage_adapter struct{ *StateTransition }

func (self dpos_storage_adapter) SubBalance(address *common.Address, b *big.Int) bool {
	if acc := self.evm_state.GetAccountConcrete(address); vm.BalanceGTE(acc, b) {
		acc.SubBalance(b)
		return true
	}
	return false
}

func (self dpos_storage_adapter) AddBalance(address *common.Address, b *big.Int) {
	self.evm_state.GetAccountConcrete(address).AddBalance(b)
}

func (self dpos_storage_adapter) Put(address *common.Address, k *common.Hash, v []byte) {
	self.evm_state.GetAccountConcrete(address).SetStateRawIrreversibly(k, v)
}

func (self dpos_storage_adapter) Get(address *common.Address, k *common.Hash, cb func([]byte)) {
	self.last_block_reader.GetAccountStorage(address, k, cb)
}

func (self dpos_storage_adapter) GetHistorical(blk_n types.BlockNum, addr *common.Address, k *common.Hash, cb func([]byte)) {
	reader := state_db.BlockReader{self.db.ReadBlock(blk_n)}
	defer reader.NotifyDone()
	reader.GetAccountStorage(addr, k, cb)
}
