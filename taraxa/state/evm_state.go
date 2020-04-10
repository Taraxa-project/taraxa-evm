package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"math/big"
)

type EVMState struct {
	in       EVMStateInput
	accounts map[common.Address]*local_account
	refund   uint64
	logs     []vm.LogRecord
	dirties  map[common.Address]*dirty_record
	journal  []journal_entry
}
type EVMStateInput interface {
	GetCode(code_hash *common.Hash) []byte
	GetAccount(addr *common.Address) (Account, bool)
	GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int
}
type local_account = struct {
	AccountChange
	storage_origin AccountStorage
	suicided       bool
	times_touched  int
}
type AccountChange = struct {
	Account
	code          []byte
	code_dirty    bool
	storage_dirty AccountStorage
}
type AccountStorage = map[common.Hash]*big.Int
type journal_entry = struct {
	dirty_addr common.Address
	read_only  bool
	revert     func()
}
type EvmStateOpts = struct {
	AccountCacheSize      int
	DirtyAccountCacheSize int
}
type dirty_record = struct {
	acc                *local_account
	times_marked_dirty int
}

func NewEVMState(src EVMStateInput, opts EvmStateOpts) (ret EVMState) {
	ret.in = src
	ret.accounts = make(map[common.Address]*local_account, opts.AccountCacheSize)
	ret.dirties = make(map[common.Address]*dirty_record, opts.DirtyAccountCacheSize)
	ret.journal = make([]journal_entry, 0, opts.DirtyAccountCacheSize*2)
	return
}

func (self *EVMState) Exist(addr common.Address) bool {
	return self.get_account(addr) != nil
}

func (self *EVMState) Empty(addr common.Address) bool {
	acc := self.get_account(addr)
	return acc == nil || acc.is_empty()
}

func (self *EVMState) GetBalance(addr common.Address) *big.Int {
	if acc := self.get_account(addr); acc != nil {
		return acc.balance
	}
	return common.Big0
}

func (self *EVMState) HasBalance(address common.Address) bool {
	return self.GetBalance(address).Sign() != 0
}

func (self *EVMState) AssertBalanceGTE(address common.Address, amount *big.Int) bool {
	return self.GetBalance(address).Cmp(amount) >= 0
}

func (self *EVMState) GetNonce(addr common.Address) uint64 {
	if acc := self.get_account(addr); acc != nil {
		return acc.nonce
	}
	return 0
}

func (self *EVMState) GetCode(addr common.Address) []byte {
	acc := self.get_account(addr)
	if acc == nil {
		return nil
	}
	if acc.code_size == 0 {
		return nil
	}
	if len(acc.code) != 0 {
		return acc.code
	}
	acc.code = self.in.GetCode(acc.code_hash)
	return acc.code
}

func (self *EVMState) GetCodeSize(addr common.Address) uint64 {
	if acc := self.get_account(addr); acc != nil {
		return acc.code_size
	}
	return 0
}

func (self *EVMState) GetCodeHash(addr common.Address) (ret common.Hash) {
	if acc := self.get_account(addr); acc != nil {
		if acc.code_size == 0 {
			ret = crypto.EmptyBytesKeccak256
		} else {
			ret = *acc.code_hash
		}
	}
	return
}

func (self *EVMState) GetState(addr common.Address, key common.Hash) *big.Int {
	if acc := self.get_account(addr); acc != nil {
		return self.get_storage(addr, acc, key)
	}
	return common.Big0
}

func (self *EVMState) GetCommittedState(addr common.Address, key common.Hash) *big.Int {
	if acc := self.get_account(addr); acc != nil {
		return self.get_origin_storage(addr, acc, key)
	}
	return common.Big0
}

func (self *EVMState) HasSuicided(addr common.Address) bool {
	if acc := self.get_account(addr); acc != nil {
		return acc.suicided
	}
	return false
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (self *EVMState) AddBalance(addr common.Address, amount *big.Int) {
	acc := self.get_or_create_account(addr)
	if amount.Sign() != 0 {
		self.set_balance(addr, acc, new(big.Int).Add(acc.balance, amount))
		return
	}
	if !acc.is_empty() || addr == ripemd {
		return
	}
	self.register_change(addr, acc, func() {
		acc.times_touched--
	})
	acc.times_touched++
}

func (self *EVMState) SubBalance(addr common.Address, amount *big.Int) {
	if acc := self.get_or_create_account(addr); amount.Sign() != 0 {
		self.set_balance(addr, acc, new(big.Int).Sub(acc.balance, amount))
	}
}

func (self *EVMState) set_balance(addr common.Address, acc *local_account, amount *big.Int) {
	balance_prev := acc.balance
	self.register_change(addr, acc, func() {
		acc.balance = balance_prev
	})
	acc.balance = amount
}

func (self *EVMState) IncrementNonce(addr common.Address) {
	acc := self.get_or_create_account(addr)
	self.register_change(addr, acc, func() {
		acc.nonce--
	})
	acc.nonce++
}

func (self *EVMState) SetCode(addr common.Address, code []byte) {
	acc := self.get_or_create_account(addr)
	assert.Holds(acc.code_size == 0)
	code_size := len(code)
	if code_size == 0 {
		return
	}
	self.register_change(addr, acc, func() {
		acc.code_dirty = false
		acc.code_hash, acc.code_size, acc.code = nil, 0, nil
	})
	acc.code_dirty = true
	acc.code_hash, acc.code_size, acc.code = util.Hash(code), uint64(code_size), code
}

func (self *EVMState) SetState(addr common.Address, key common.Hash, value *big.Int) {
	acc := self.get_or_create_account(addr)
	prev := self.get_storage(addr, acc, key)
	if prev.Cmp(value) == 0 {
		return
	}
	self.register_change(addr, acc, func() {
		acc.storage_dirty[key] = prev
	})
	if acc.storage_dirty == nil {
		acc.storage_dirty = make(AccountStorage)
	}
	acc.storage_dirty[key] = new(big.Int).Set(value)
}

func (self *EVMState) Suicide(addr common.Address, newAddr common.Address) {
	acc := self.get_account(addr)
	if acc == nil {
		self.AddBalance(newAddr, common.Big0)
		return
	}
	self.AddBalance(newAddr, acc.balance)
	suicided_prev, balance_prev := acc.suicided, acc.balance
	self.register_change(addr, acc, func() {
		acc.suicided, acc.balance = suicided_prev, balance_prev
	})
	acc.suicided, acc.balance = true, common.Big0
}

func (self *EVMState) CreateAccount(addr common.Address) {
	prev := self.get_account(addr)
	if prev == nil {
		self.create_account(addr)
		return
	}
	prev_val := *prev
	self.register_change(addr, prev, func() {
		*prev = prev_val
	})
	*prev = local_account{}
	prev.balance = prev_val.balance
}

func (self *EVMState) get_or_create_account(addr common.Address) *local_account {
	if acc := self.get_account(addr); acc != nil {
		return acc
	}
	return self.create_account(addr)
}

func (self *EVMState) get_account(addr common.Address) *local_account {
	if acc, present := self.accounts[addr]; present {
		return acc
	}
	new_acc, exists := new(local_account), false
	if new_acc.Account, exists = self.in.GetAccount(&addr); exists {
		self.accounts[addr] = new_acc
		return new_acc
	}
	self.accounts[addr] = nil
	return nil
}

func (self *EVMState) create_account(addr common.Address) *local_account {
	new := new_account()
	self.register_change(addr, new, func() {
		self.accounts[addr] = nil
		delete(self.dirties, addr)
	})
	self.accounts[addr] = new
	return new
}

func new_account() *local_account {
	ret := new(local_account)
	ret.balance = common.Big0
	return ret
}

func (self *EVMState) get_storage(addr common.Address, acc *local_account, key common.Hash) *big.Int {
	if value, present := acc.storage_dirty[key]; present {
		return value
	}
	return self.get_origin_storage(addr, acc, key)
}

func (self *EVMState) get_origin_storage(addr common.Address, acc *local_account, key common.Hash) *big.Int {
	if ret, present := acc.storage_origin[key]; present {
		return ret
	}
	if len(acc.storage_root_hash) == 0 {
		return common.Big0
	}
	ret := self.in.GetAccountStorage(&addr, &key)
	if acc.storage_origin == nil {
		acc.storage_origin = make(AccountStorage)
	}
	acc.storage_origin[key] = ret
	return ret
}

func (self *EVMState) AddLog(log vm.LogRecord) {
	lastpos := len(self.logs) - 1
	self.register_change_r_only(func() {
		self.logs = self.logs[:lastpos]
	})
	self.logs = append(self.logs, log)
}

func (self *EVMState) GetLogs() []vm.LogRecord {
	return self.logs
}

func (self *EVMState) AddRefund(gas uint64) {
	prev := self.refund
	self.register_change_r_only(func() {
		self.refund = prev
	})
	self.refund += gas
}

func (self *EVMState) SubRefund(gas uint64) {
	if gas > self.refund {
		panic("Refund counter below zero")
	}
	prev := self.refund
	self.register_change_r_only(func() {
		self.refund = prev
	})
	self.refund -= gas
}

func (self *EVMState) GetRefund() uint64 {
	return self.refund
}

func (self *EVMState) Snapshot() int {
	return len(self.journal)
}

func (self *EVMState) RevertToSnapshot(snapshot int) {
	for i := len(self.journal) - 1; i >= snapshot; i-- {
		change := &self.journal[i]
		if !change.read_only {
			self.dirties[change.dirty_addr].times_marked_dirty--
		}
		change.revert()
	}
	self.journal = self.journal[:snapshot]
}

func (self *EVMState) register_change(addr common.Address, acc *local_account, revert func()) {
	dirty_rec := self.dirties[addr]
	if dirty_rec == nil {
		dirty_rec = &dirty_record{acc: acc}
		self.dirties[addr] = dirty_rec
	}
	dirty_rec.times_marked_dirty++
	self.journal = append(self.journal, journal_entry{dirty_addr: addr, revert: revert})
}

func (self *EVMState) register_change_r_only(revert func()) {
	self.journal = append(self.journal, journal_entry{read_only: true, revert: revert})
}

type EVMStateOutput = struct {
	OnAccountChanged func(address common.Address, change AccountChange)
	OnAccountDeleted func(address common.Address)
}

func (self *EVMState) Commit(delete_empty_accounts bool, sink EVMStateOutput) {
	dirties := self.dirties
	self.dirties = make(map[common.Address]*dirty_record)
	self.journal, self.refund, self.logs = self.journal[:0], 0, self.logs[:0]
	for addr, dirty_rec := range dirties {
		if dirty_rec.times_marked_dirty == 0 {
			continue
		}
		acc := dirty_rec.acc
		touched_only := dirty_rec.times_marked_dirty == acc.times_touched
		acc.times_touched = 0
		if acc.suicided || delete_empty_accounts && acc.is_empty() {
			sink.OnAccountDeleted(addr)
			self.accounts[addr] = nil
			continue
		}
		if touched_only {
			continue
		}
		sink.OnAccountChanged(addr, acc.AccountChange)
		for k, v := range acc.storage_dirty {
			acc.storage_origin[k] = v
		}
		acc.storage_dirty = nil
	}
}
