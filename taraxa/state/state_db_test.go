package state

import (
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
)

func TestStateDBTransientStorage(t *testing.T) {
	var state state_evm.TransitionState
	state.Init(state_evm.Opts{
		NumTransactionsToBuffer: 1,
	})

	key := common.Hash{0x01}
	value := common.Hash{0x02}
	addr := common.Address{}

	state.SetTransientState(&addr, key, value)
	// the retrieved value should equal what was set
	if got := state.GetTransientState(&addr, key); got != value {
		t.Fatalf("transient storage mismatch: have %x, want %x", got, value)
	}

	// revert the transient state being set and then check that the
	// value is now the empty hash
	state.CommitTransaction(nil)
	if got, exp := state.GetTransientState(&addr, key), (common.Hash{}); exp != got {
		t.Fatalf("transient storage mismatch: have %x, want %x", got, exp)
	}
}
