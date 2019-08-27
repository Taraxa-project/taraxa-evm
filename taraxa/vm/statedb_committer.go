package vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

type StateDBCommitter struct {
	inbox  chan state.StateChange
	outbox chan common.Hash
}

func LaunchStateDBCommitter(numStateChanges int, newStateDB StateDBFactory, commit CommitStrategy) *StateDBCommitter {
	this := &StateDBCommitter{
		make(chan state.StateChange, numStateChanges),
		make(chan common.Hash, 1),
	}
	go func() {
		var root common.Hash
		stateDB := newStateDB()
		for i := 0; i < cap(this.inbox); i++ {
			stateChange, ok := <-this.inbox
			if !ok {
				close(this.outbox)
				return
			}
			stateDB.Merge(stateChange)
			root = commit(stateDB)
		}
		this.outbox <- root
	}()
	return this
}

func (this *StateDBCommitter) Halt() error {
	return concurrent.TryClose(this.inbox)
}

func (this *StateDBCommitter) Submit(change state.StateChange) error {
	return concurrent.TrySend(this.inbox, change)
}

func (this *StateDBCommitter) AwaitResult() (ret common.Hash, ok bool) {
	ret, ok = <-this.outbox
	return
}
