package state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"math/big"
)

type TransientState struct {
	BalanceDeltas map[common.Address]*big.Int
	NonceDeltas   map[common.Address]uint64
}

func NewTransientState() *TransientState {
	ret := new(TransientState)
	ret.BalanceDeltas = make(map[common.Address]*big.Int)
	ret.NonceDeltas = make(map[common.Address]uint64)
	return ret
}

func (this *TransientState) Clone() *TransientState {
	ret := NewTransientState()
	for k, v := range this.BalanceDeltas {
		ret.BalanceDeltas[k] = v
	}
	for k, v := range this.NonceDeltas {
		ret.NonceDeltas[k] = v
	}
	return ret
}
