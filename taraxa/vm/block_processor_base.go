package vm

import (
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type blockProcessorBase struct {
	*VM
	*StateTransitionRequest
	err util.ErrorBarrier
}

func (this *blockProcessorBase) ValidateNonces() error {
	stateDB, err := state.New(this.BaseStateRoot, this.ReadDB)
	this.err.CheckIn(err)
	var validator = NonceValidator{stateDB.GetNonce}
	for _, tx := range this.Block.Transactions {
		if err := validator.Append(tx.From, tx.Nonce); err != nil {
			return err
		}
	}
	return nil
}
