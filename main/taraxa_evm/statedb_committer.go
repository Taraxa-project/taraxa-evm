package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type StateDbCommitter struct {
	stateDBDeltasChan chan *state.StateChange
	finalRootChan     chan common.Hash
}

type CommitFunction func(StateDB) (common.Hash, error)

func LaunchStateDBCommitter(
	expectedNumberOfStatechanges int,
	stateRoot common.Hash,
	db state.Database,
	errBarrier *util.ErrorBarrier,
	commit CommitFunction,
) *StateDbCommitter {
	stateDBDeltasChan := make(chan *state.StateChange, expectedNumberOfStatechanges)
	finalRootChan := make(chan common.Hash, 1)
	go func() {
		defer util.Recover(errBarrier.Catch(func(error) {
			close(finalRootChan)
		}))
		stateDB, err := state.New(stateRoot, db)
		errBarrier.CheckIn(err)
		lastRoot := stateRoot
		for i := 0; i < cap(stateDBDeltasChan); i++ {
			txStateDB := <-stateDBDeltasChan
			errBarrier.CheckIn()
			stateDB.MergeChanges(txStateDB)
			lastRoot, err = commit(stateDB)
			errBarrier.CheckIn(err)
		}
		finalRootChan <- lastRoot
	}()
	return &StateDbCommitter{stateDBDeltasChan, finalRootChan}
}

func (this *StateDbCommitter) RequestCommit(change *state.StateChange) {
	this.stateDBDeltasChan <- change
}

func (this *StateDbCommitter) AwaitFinalRoot() common.Hash {
	return <-this.finalRootChan
}
