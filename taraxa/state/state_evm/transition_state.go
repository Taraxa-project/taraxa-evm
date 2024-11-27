package state_evm

import (
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

type EVMStateFace interface {
	vm.State
	GetAccountConcrete(*common.Address) *Account
	GetAccountStorage(addr *common.Address, k *common.Hash, cb func([]byte))
	GetAccountStorageFromDB(addr *common.Address, k *common.Hash, cb func([]byte))
	In() Input
	RegisterChange(revert func())
	DeleteAccount(acc *Account)
}

type EVMStateAccountHeader struct {
	host            EVMStateFace
	in_dirties      bool
	in_curr_version bool
	loaded_from_db  bool
	deleted         bool
}
type Accounts = []*Account
type Opts struct {
	NumTransactionsToBuffer uint64
}

type TransitionState struct {
	in                            Input
	accounts                      AccountMap
	accounts_in_curr_ver_original Accounts
	accounts_in_curr_ver          Accounts
	reverts_original, reverts     []func()
	dirties_original              Accounts
	dirties                       Accounts
	logs                          []vm.LogRecord
	refund                        uint64
	transientStorage              state_db.TransientStorage
}

func (ts *TransitionState) In() Input {
	return ts.in
}

func (ts *TransitionState) Init(opts Opts) {
	if opts.NumTransactionsToBuffer == 0 {
		opts.NumTransactionsToBuffer = 1
	}
	ts.accounts.Init(AccountMapOptions{opts.NumTransactionsToBuffer * 32, 4})
	ts.accounts_in_curr_ver_original = make(Accounts, 0, 256)
	ts.accounts_in_curr_ver = ts.accounts_in_curr_ver_original
	ts.reverts_original = make([]func(), 0, 1024) // 8KB
	ts.reverts = ts.reverts_original
	ts.dirties_original = make(Accounts, 0, opts.NumTransactionsToBuffer*16)
	ts.dirties = ts.dirties_original
}

func (ts *TransitionState) SetInput(in Input) {
	ts.in = in
}

func (ts *TransitionState) GetAccount(addr *common.Address) vm.StateAccount {
	return ts.GetAccountConcrete(addr)
}

func (ts *TransitionState) DeleteAccount(acc *Account) {
	ts.accounts.Delete(acc)
}

func (ts *TransitionState) GetAccountConcrete(addr *common.Address) *Account {
	acc, was_present := ts.accounts.GetOrNew(addr)
	if !acc.in_curr_version {
		acc.in_curr_version = true
		ts.accounts_in_curr_ver = append(ts.accounts_in_curr_ver, acc)
	}
	if was_present {
		return acc
	}
	acc.host = ts
	ts.in.GetAccount(addr, func(db_acc state_db.Account) {
		acc.AccountBody = &AccountBody{AccountChange: AccountChange{Account: db_acc}}
		acc.loaded_from_db = true
	})
	return acc
}

func (ts *TransitionState) GetAccountStorage(addr *common.Address, k *common.Hash, cb func([]byte)) {
	ts.GetAccountStorageFromDB(addr, k, cb)
}

func (ts *TransitionState) GetAccountStorageFromDB(addr *common.Address, k *common.Hash, cb func([]byte)) {
	ts.in.GetAccountStorage(addr, k, cb)
}

func (ts *TransitionState) AddLog(log vm.LogRecord) {
	pos := len(ts.logs)
	ts.RegisterChange(func() {
		ts.logs = ts.logs[:pos]
	})
	ts.logs = append(ts.logs, log)
}

func (ts *TransitionState) GetLogs() []vm.LogRecord {
	return ts.logs
}

func (ts *TransitionState) AddRefund(gas uint64) {
	prev := ts.refund
	ts.RegisterChange(func() {
		ts.refund = prev
	})
	ts.refund += gas
}

func (ts *TransitionState) SubRefund(gas uint64) {
	if gas > ts.refund {
		panic("Refund counter below zero")
	}
	prev := ts.refund
	ts.RegisterChange(func() {
		ts.refund = prev
	})
	ts.refund -= gas
}

func (ts *TransitionState) GetRefund() uint64 {
	return ts.refund
}

func (ts *TransitionState) Snapshot() int {
	return len(ts.reverts)
}

func (ts *TransitionState) RevertToSnapshot(snapshot int) {
	for i := len(ts.reverts) - 1; i >= snapshot; i-- {
		ts.reverts[i]()
	}
	ts.reverts = ts.reverts[:snapshot]
}

func (ts *TransitionState) RegisterChange(revert func()) {
	ts.reverts = append(ts.reverts, revert)
}

func (ts *TransitionState) CommitTransaction(db_writer Output) {
	for _, acc := range ts.accounts_in_curr_ver {
		acc.in_curr_version = false
		if acc.deleted {
			continue
		}
		status := acc.flush(db_writer)
		acc.deleted = status == deleted
		if !acc.in_dirties && (status == updated || acc.deleted && acc.loaded_from_db) {
			acc.in_dirties = true
			ts.dirties = append(ts.dirties, acc)
		}
		if !acc.in_dirties {
			acc.unload()
		}
	}
	bin.ZFill_2(
		unsafe.Pointer(&ts.accounts_in_curr_ver_original),
		len(ts.accounts_in_curr_ver),
		unsafe.Sizeof(ts.accounts_in_curr_ver[0]))
	ts.accounts_in_curr_ver = ts.accounts_in_curr_ver_original
	bin.ZFill_2(
		unsafe.Pointer(&ts.reverts_original),
		len(ts.reverts),
		unsafe.Sizeof(ts.reverts[0]))
	ts.reverts = ts.reverts_original
	ts.logs = nil
	ts.refund = 0
	// Reset transient storage
	ts.transientStorage = nil
}

func (ts *TransitionState) Commit() {
	for _, acc := range ts.dirties {
		if !acc.deleted {
			acc.sink.Commit()
		}
		acc.unload()
	}
	bin.ZFill_2(
		unsafe.Pointer(&ts.dirties_original),
		len(ts.dirties),
		unsafe.Sizeof(ts.dirties[0]))
	ts.dirties = ts.dirties_original
}

func (ts *TransitionState) initTransientState() {
	if ts.transientStorage == nil {
		ts.transientStorage = make(state_db.TransientStorage)
	}
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (ts *TransitionState) SetTransientState(addr *common.Address, key, value common.Hash) {
	ts.initTransientState()
	prev := ts.GetTransientState(addr, key)
	if prev == value {
		return
	}
	ts.setTransientState(addr, key, value)
}

// setTransientState is a lower level setter for transient storage. It
// is called during a revert to prevent modifications to the journal.
func (ts *TransitionState) setTransientState(addr *common.Address, key, value common.Hash) {
	ts.transientStorage.Set(*addr, key, value)
}

// GetTransientState gets transient storage for a given account.
func (ts *TransitionState) GetTransientState(addr *common.Address, key common.Hash) common.Hash {
	ts.initTransientState()
	return ts.transientStorage.Get(*addr, key)
}
