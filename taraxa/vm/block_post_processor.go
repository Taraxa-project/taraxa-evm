package vm

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

type BlockPostProcessor struct {
	inbox  []chan *TransactionResultWithStateChange
	outbox chan *StateTransitionReceipt
}

func LaunchBlockPostProcessor(block *Block, newStateDB StateDBFactory, onErr util.ErrorHandler) *BlockPostProcessor {
	inbox := make([]chan *TransactionResultWithStateChange, len(block.Transactions))
	outbox := make(chan *StateTransitionReceipt, 1)
	for txId := range block.Transactions {
		inbox[txId] = make(chan *TransactionResultWithStateChange, 1)
	}
	go func() {
		result := new(StateTransitionReceipt)
		var err util.ErrorBarrier
		defer util.Recover(err.Catch(func(e error) {
			close(outbox)
			onErr(e)
		}))
		stateDB := newStateDB()
		gasPool := new(core.GasPool).AddGas(block.GasLimit)
		for txId, channel := range inbox {
			tx := block.Transactions[txId]
			err.CheckIn(gasPool.SubGas(tx.GasLimit))
			request, ok := <-channel
			if !ok {
				close(outbox)
				return
			}
			concurrent.TryClose(channel)
			stateDB.Merge(request.StateChange)
			for k := range request.StateChange {
				if stateDB.GetBalance(k).Sign() < 0 {
					//err.CheckIn(vm.ErrInsufficientBalance)
				}
			}
			gasPool.AddGas(tx.GasLimit - request.GasUsed)
			result.UsedGas += request.GasUsed
			ethReceipt := types.NewReceipt(nil, request.ContractErr != nil, result.UsedGas)
			ethReceipt.GasUsed = request.GasUsed
			ethReceipt.TxHash = tx.Hash;
			ethReceipt.Logs = request.Logs
			ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
			if tx.To == nil {
				ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, tx.Nonce)
			}
			result.AllLogs = append(result.AllLogs, ethReceipt.Logs...)
			result.Receipts = append(result.Receipts, &TaraxaReceipt{
				request.EVMReturnValue,
				ethReceipt,
				request.ContractErr,
			})
		}
		outbox <- result
	}()
	return &BlockPostProcessor{inbox, outbox}
}

func (this *BlockPostProcessor) Halt() error {
	closedAtLeastOne := false
	for _, channel := range this.inbox {
		closedAtLeastOne = concurrent.TryClose(channel) == nil || closedAtLeastOne
	}
	if closedAtLeastOne {
		return nil
	}
	return errors.New("Already closed")
}

func (this *BlockPostProcessor) Submit(request *TransactionResultWithStateChange) error {
	return concurrent.TrySend(this.inbox[request.TxId], request)
}

func (this *BlockPostProcessor) AwaitResult() (ret *StateTransitionReceipt, ok bool) {
	ret, ok = <-this.outbox
	return
}