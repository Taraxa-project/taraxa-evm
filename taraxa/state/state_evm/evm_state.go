package state_evm

import (
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigconv"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"

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
	logs                          []vm.LogRecord
	refund                        uint64
	version                       uint64
	dirties                       *Dirties
	bigconv                       bigconv.BigConv
}
type EVMStateAccountHeader struct {
	host                 *EVMState
	upd_version          uint64
	read_in_curr_version bool
	loaded_from_db       bool //TODO refactor into state machine
	deleted              bool //TODO refactor into state machine
}
type Accounts = []*Account
type CacheOpts struct {
	AccountBufferSize uint32
	RevertLogSize     uint32
}

func (self *EVMState) Init(in Input, cache_opts CacheOpts) {
	self.in = in
	// TODO think about better config
	self.accounts.Init(AccountMapOptions{cache_opts.AccountBufferSize, 4})
	self.accounts_in_curr_ver_original = make(Accounts, 0, cache_opts.AccountBufferSize/8)
	self.accounts_in_curr_ver = self.accounts_in_curr_ver_original
	self.reverts_original = make([]func(), 0, cache_opts.RevertLogSize)
	self.reverts = self.reverts_original
	self.version = 1
}

func (self *EVMState) GetAccount(addr *common.Address) vm.StateAccount {
	return self.GetAccountConcrete(addr)
}

func (self *EVMState) GetAccountConcrete(addr *common.Address) *Account {
	acc, was_present := self.accounts.GetOrNew(addr)
	if !acc.read_in_curr_version {
		acc.read_in_curr_version = true
		self.accounts_in_curr_ver = append(self.accounts_in_curr_ver, acc)
	}
	if was_present {
		return acc
	}
	acc.host = self
	self.in.GetRawAccount(addr, func(bytes []byte) {
		acc.AccountBody = new(AccountBody)
		acc.DecodeStorageRepr(bytes)
		acc.loaded_from_db = true
	})
	return acc
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

func (self *EVMState) Checkpoint(sink Sink, eip158 bool) {
	if self.dirties == nil {
		self.dirties = new(Dirties).Init(0)
	}
	num_accs_to_clean := 0
	for i, acc := range self.accounts_in_curr_ver {
		acc.read_in_curr_version = false
		if acc.deleted {
			continue
		}
		if status := acc.flush(sink, eip158); status == updated {
			acc.upd_version = self.version
			self.dirties.append(acc)
		} else if acc.deleted = status == deleted; acc.deleted || status == unmodified && acc.upd_version == 0 {
			if status == unmodified || !acc.loaded_from_db {
				if i != num_accs_to_clean {
					self.accounts_in_curr_ver[num_accs_to_clean] = acc
				}
				num_accs_to_clean++
			} else {
				self.dirties.append(acc)
			}
		}
	}
	for i := 0; i < num_accs_to_clean; i++ {
		self.accounts_in_curr_ver[i].unload()
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
	self.version++
}

func (self *EVMState) Commit() (ret *Dirties) {
	self.dirties.set_commit_version(self.version)
	ret, self.dirties = self.dirties, nil
	return
}

func (self *EVMState) Cleanup(b *Dirties) {
	assert.Holds(self.dirties == nil, "cleanup is impossible with uncommitted checkpoints")
	b.cleanup()
	self.dirties = b
}
