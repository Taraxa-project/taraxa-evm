package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
)

type NonceValidator struct {
	GetNonce func(common.Address) uint64
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

func ValidateNonces(transactions []*vm.Transaction, GetNonce func(common.Address) uint64) error {
	var validator = NonceValidator{GetNonce: GetNonce}
	for _, tx := range transactions {
		if err := validator.Append(tx.From, uint64(tx.Nonce)); err != nil {
			return err
		}
	}
	return nil
}
