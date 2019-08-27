package vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
)

type NonceValidator struct {
	GetNonce func(address common.Address) uint64
	state    map[common.Address]uint64
}

func (this *NonceValidator) Append(address common.Address, expectedNonce uint64) error {
	nonce, hasBeenLoaded := this.state[address]
	if !hasBeenLoaded {
		nonce = this.GetNonce(address)
		this.state[address] = nonce
	}
	if nonce == expectedNonce {
		this.state[address]++
		return nil
	}
	if nonce < expectedNonce {
		return core.ErrNonceTooHigh
	}
	return core.ErrNonceTooLow
}
