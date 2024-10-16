package state_evm

import (
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigconv"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

type EVMState struct {
	in                            Input
	accounts                      AccountMap
	accounts_in_curr_ver_original Accounts
	accounts_in_curr_ver          Accounts
	reverts_original, reverts     []func()
	dirties_original              Accounts
	dirties                       Accounts
	logs                          []vm.LogRecord
	refund                        uint64
	bigconv                       bigconv.BigConv
	transientStorage              state_db.TransientStorage
}
type EVMStateAccountHeader struct {
	host            *EVMState
	in_dirties      bool
	in_curr_version bool
	loaded_from_db  bool
	deleted         bool
}
type Accounts = []*Account
type Opts struct {
	NumTransactionsToBuffer uint64
}

func (self *EVMState) Init(opts Opts) {
	if opts.NumTransactionsToBuffer == 0 {
		opts.NumTransactionsToBuffer = 1
	}
	self.accounts.Init(AccountMapOptions{opts.NumTransactionsToBuffer * 32, 4})
	self.accounts_in_curr_ver_original = make(Accounts, 0, 256)
	self.accounts_in_curr_ver = self.accounts_in_curr_ver_original
	self.reverts_original = make([]func(), 0, 1024) // 8KB
	self.reverts = self.reverts_original
	self.dirties_original = make(Accounts, 0, opts.NumTransactionsToBuffer*16)
	self.dirties = self.dirties_original
}

func (self *EVMState) SetInput(in Input) {
	self.in = in
}

func (self *EVMState) GetAccount(addr *common.Address) vm.StateAccount {
	return self.GetAccountConcrete(addr)
}

func (self *EVMState) GetAccountConcrete(addr *common.Address) *Account {
	acc, was_present := self.accounts.GetOrNew(addr)
	if !acc.in_curr_version {
		acc.in_curr_version = true
		self.accounts_in_curr_ver = append(self.accounts_in_curr_ver, acc)
	}
	if was_present {
		return acc
	}
	acc.host = self
	self.in.GetAccount(addr, func(db_acc state_db.Account) {
		acc.AccountBody = &AccountBody{AccountChange: AccountChange{Account: db_acc}}
		acc.loaded_from_db = true
	})
	return acc
}

func (self *EVMState) GetAccountStorageFromDB(addr *common.Address, k *common.Hash, cb func([]byte)) {
	self.in.GetAccountStorage(addr, k, cb)
}

func (self *EVMState) AddLog(log vm.LogRecord) {
	pos := len(self.logs)
	self.register_change(func() {
		self.logs = self.logs[:pos]
	})
	self.logs = append(self.logs, log)
}

func (self *EVMState) GetLogs() []vm.LogRecord {
	return self.logs
}

func (self *EVMState) AddRefund(gas uint64) {
	prev := self.refund
	self.register_change(func() {
		self.refund = prev
	})
	self.refund += gas
}

func (self *EVMState) SubRefund(gas uint64) {
	if gas > self.refund {
		panic("Refund counter below zero")
	}
	prev := self.refund
	self.register_change(func() {
		self.refund = prev
	})
	self.refund -= gas
}

func (self *EVMState) GetRefund() uint64 {
	return self.refund
}

func (self *EVMState) Snapshot() int {
	return len(self.reverts)
}

func (self *EVMState) RevertToSnapshot(snapshot int) {
	for i := len(self.reverts) - 1; i >= snapshot; i-- {
		self.reverts[i]()
	}
	self.reverts = self.reverts[:snapshot]
}

func (self *EVMState) register_change(revert func()) {
	self.reverts = append(self.reverts, revert)
}

func (self *EVMState) CommitTransaction(db_writer Output) {
	for _, acc := range self.accounts_in_curr_ver {
		acc.in_curr_version = false
		if acc.deleted {
			continue
		}
		status := acc.flush(db_writer)
		acc.deleted = status == deleted
		if !acc.in_dirties && (status == updated || acc.deleted && acc.loaded_from_db) {
			acc.in_dirties = true
			self.dirties = append(self.dirties, acc)
		}
		if !acc.in_dirties {
			acc.unload()
		}
	}
	bin.ZFill_2(
		unsafe.Pointer(&self.accounts_in_curr_ver_original),
		len(self.accounts_in_curr_ver),
		unsafe.Sizeof(self.accounts_in_curr_ver[0]))
	self.accounts_in_curr_ver = self.accounts_in_curr_ver_original
	bin.ZFill_2(
		unsafe.Pointer(&self.reverts_original),
		len(self.reverts),
		unsafe.Sizeof(self.reverts[0]))
	self.reverts = self.reverts_original
	self.logs = nil
	self.refund = 0
	// Reset transient storage
	self.transientStorage = nil
}

func (self *EVMState) Commit() {
	for _, acc := range self.dirties {
		if !acc.deleted {
			acc.sink.Commit()
		}
		acc.unload()
	}
	bin.ZFill_2(
		unsafe.Pointer(&self.dirties_original),
		len(self.dirties),
		unsafe.Sizeof(self.dirties[0]))
	self.dirties = self.dirties_original
}

func (self *EVMState) initTransientState() {
	if self.transientStorage == nil {
		self.transientStorage = make(state_db.TransientStorage)
	}
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (self *EVMState) SetTransientState(addr *common.Address, key, value common.Hash) {
	self.initTransientState()
	prev := self.GetTransientState(addr, key)
	if prev == value {
		return
	}
	self.setTransientState(addr, key, value)
}

// setTransientState is a lower level setter for transient storage. It
// is called during a revert to prevent modifications to the journal.
func (self *EVMState) setTransientState(addr *common.Address, key, value common.Hash) {
	self.transientStorage.Set(*addr, key, value)
}

// GetTransientState gets transient storage for a given account.
func (self *EVMState) GetTransientState(addr *common.Address, key common.Hash) common.Hash {
	self.initTransientState()
	return self.transientStorage.Get(*addr, key)
}
