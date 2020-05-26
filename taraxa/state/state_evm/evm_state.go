package state_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"math/big"
)

type EVMState struct {
	in            Input
	refund        uint64
	logs          []vm.LogRecord
	accounts_keys []common.Address
	accounts      map[common.Address]*local_account
	dirties       []dirty_record
	reverts       []func()
}
type dirty_record struct {
	addr common.Address
	acc  *local_account
}
type local_account struct {
	AccountChange
	storage_origin AccountStorage
	suicided       bool
	times_touched  uint32
	times_dirtied  uint32
	in_dirties     bool
}

type CacheOpts struct {
	AccountsPrealloc      uint32
	DirtyAccountsPrealloc uint32
}

func (self *EVMState) Init(in Input, cache_opts CacheOpts) {
	self.in = in
	self.accounts_keys = make([]common.Address, 0, cache_opts.AccountsPrealloc)
	self.accounts = make(map[common.Address]*local_account, cache_opts.AccountsPrealloc)
	self.dirties = make([]dirty_record, 0, cache_opts.DirtyAccountsPrealloc)
	self.reverts = make([]func(), 0, cache_opts.DirtyAccountsPrealloc*3)
}

func (self *EVMState) Reset() {
	self.reset_state_change()
	for _, addr := range self.accounts_keys {
		delete(self.accounts, addr)
	}
	self.accounts_keys = self.accounts_keys[:0]
}

func (self *EVMState) reset_state_change() {
	self.dirties, self.reverts, self.logs, self.refund = self.dirties[:0], self.reverts[:0], self.logs[:0], 0
}

func (self *EVMState) Exist(addr common.Address) bool {
	return self.get_account(addr) != nil
}

func (self *EVMState) Empty(addr common.Address) bool {
	acc := self.get_account(addr)
	return acc == nil || is_empty(acc)
}

func (self *EVMState) GetBalance(addr common.Address) *big.Int {
	if acc := self.get_account(addr); acc != nil {
		return acc.Balance
	}
	return common.Big0
}

func (self *EVMState) HasBalance(addr common.Address) bool {
	return self.GetBalance(addr).Sign() != 0
}

func (self *EVMState) AssertBalanceGTE(addr common.Address, amount *big.Int) bool {
	return amount.Sign() == 0 || self.GetBalance(addr).Cmp(amount) >= 0
}

func (self *EVMState) GetNonce(addr common.Address) uint64 {
	if acc := self.get_account(addr); acc != nil {
		return acc.Nonce
	}
	return 0
}

func (self *EVMState) GetCode(addr common.Address) []byte {
	acc := self.get_account(addr)
	if acc == nil {
		return nil
	}
	if acc.CodeSize == 0 {
		return nil
	}
	if len(acc.Code) != 0 {
		return acc.Code
	}
	acc.Code = self.in.GetCode(acc.CodeHash)
	return acc.Code
}

func (self *EVMState) GetCodeSize(addr common.Address) uint64 {
	if acc := self.get_account(addr); acc != nil {
		return acc.CodeSize
	}
	return 0
}

func (self *EVMState) GetCodeHash(addr common.Address) (ret common.Hash) {
	if acc := self.get_account(addr); acc != nil {
		if acc.CodeSize == 0 {
			ret = crypto.EmptyBytesKeccak256
		} else {
			ret = *acc.CodeHash
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

var ripemd_addr = common.BytesToAddress([]byte{3})

func (self *EVMState) AddBalance(addr common.Address, amount *big.Int) {
	acc := self.get_or_create_account(addr)
	if amount.Sign() != 0 {
		self.set_balance(addr, acc, new(big.Int).Add(acc.Balance, amount))
		return
	}
	if !is_empty(acc) {
		return
	}
	self.add_acc_revert(addr, acc, func() {
		acc.times_touched--
	})
	acc.times_touched++
	if addr == ripemd_addr {
		acc.times_dirtied++
	}
}

func (self *EVMState) SubBalance(addr common.Address, amount *big.Int) {
	acc := self.get_or_create_account(addr)
	if amount.Sign() != 0 {
		self.set_balance(addr, acc, new(big.Int).Sub(acc.Balance, amount))
	}
}

func (self *EVMState) set_balance(addr common.Address, acc *local_account, amount *big.Int) {
	balance_prev := acc.Balance
	self.add_acc_revert(addr, acc, func() {
		acc.Balance = balance_prev
	})
	acc.Balance = amount
}

func (self *EVMState) IncrementNonce(addr common.Address) {
	acc := self.get_or_create_account(addr)
	self.add_acc_revert(addr, acc, func() {
		acc.Nonce--
	})
	acc.Nonce++
}

func (self *EVMState) SetCode(addr common.Address, code []byte) {
	acc := self.get_or_create_account(addr)
	assert.Holds(acc.CodeSize == 0)
	code_size := len(code)
	if code_size == 0 {
		return
	}
	self.add_acc_revert(addr, acc, func() {
		acc.CodeDirty, acc.CodeHash, acc.CodeSize, acc.Code = false, nil, 0, nil
	})
	acc.CodeDirty, acc.CodeHash, acc.CodeSize, acc.Code = true, keccak256.Hash(code), uint64(code_size), code
}

func (self *EVMState) SetState(addr common.Address, key common.Hash, value *big.Int) {
	acc := self.get_or_create_account(addr)
	prev := self.get_storage(addr, acc, key)
	if prev.Cmp(value) == 0 {
		return
	}
	self.add_acc_revert(addr, acc, func() {
		acc.StorageDirty[key] = prev
	})
	if acc.StorageDirty == nil {
		acc.StorageDirty = make(AccountStorage)
	}
	acc.StorageDirty[key] = new(big.Int).Set(value)
}

func (self *EVMState) Suicide(addr common.Address, newAddr common.Address) {
	acc := self.get_account(addr)
	if acc == nil {
		self.AddBalance(newAddr, common.Big0)
		return
	}
	self.AddBalance(newAddr, acc.Balance)
	suicided_prev, balance_prev := acc.suicided, acc.Balance
	self.add_acc_revert(addr, acc, func() {
		acc.suicided, acc.Balance = suicided_prev, balance_prev
	})
	acc.suicided, acc.Balance = true, common.Big0
}

func (self *EVMState) get_or_create_account(addr common.Address) *local_account {
	if acc := self.get_account(addr); acc != nil {
		return acc
	}
	new := new(local_account)
	new.Balance = common.Big0
	self.add_acc_revert(addr, new, func() {
		self.accounts[addr], new.in_dirties = nil, false
	})
	self.accounts[addr] = new
	return new
}

func (self *EVMState) get_account(addr common.Address) *local_account {
	if acc, present := self.accounts[addr]; present {
		return acc
	}
	self.accounts_keys = append(self.accounts_keys, addr)
	if acc, exists := self.in.GetAccount(&addr); exists {
		new_acc := new(local_account)
		new_acc.Account = acc
		self.accounts[addr] = new_acc
		return new_acc
	}
	self.accounts[addr] = nil
	return nil
}

func (self *EVMState) get_storage(addr common.Address, acc *local_account, key common.Hash) *big.Int {
	if value, present := acc.StorageDirty[key]; present {
		return value
	}
	return self.get_origin_storage(addr, acc, key)
}

func (self *EVMState) get_origin_storage(addr common.Address, acc *local_account, key common.Hash) *big.Int {
	if ret, present := acc.storage_origin[key]; present {
		return ret
	}
	if acc.StorageRootHash == nil {
		return common.Big0
	}
	ret := self.in.GetAccountStorage(&addr, &key)
	if acc.storage_origin == nil {
		acc.storage_origin = make(AccountStorage)
	}
	acc.storage_origin[key] = ret
	return ret
}

func is_empty(acc *local_account) bool {
	return acc.Nonce == 0 && acc.Balance.Sign() == 0 && acc.CodeSize == 0
}

func (self *EVMState) AddLog(log vm.LogRecord) {
	pos := len(self.logs)
	self.add_revert(func() {
		self.logs = self.logs[:pos]
	})
	self.logs = append(self.logs, log)
}

func (self *EVMState) GetLogs() []vm.LogRecord {
	return self.logs
}

func (self *EVMState) AddRefund(gas uint64) {
	prev := self.refund
	self.add_revert(func() {
		self.refund = prev
	})
	self.refund += gas
}

func (self *EVMState) SubRefund(gas uint64) {
	if gas > self.refund {
		panic("Refund counter below zero")
	}
	prev := self.refund
	self.add_revert(func() {
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

func (self *EVMState) add_acc_revert(addr common.Address, acc *local_account, revert func()) {
	if !acc.in_dirties {
		self.dirties = append(self.dirties, dirty_record{addr, acc})
		acc.in_dirties = true
	}
	acc.times_dirtied++
	self.add_revert(func() {
		acc.times_dirtied--
		revert()
	})
}

func (self *EVMState) add_revert(revert func()) {
	self.reverts = append(self.reverts, revert)
}

func (self *EVMState) Commit(delete_empty_accounts bool, out Output) {
	defer self.reset_state_change()
	for _, rec := range self.dirties {
		acc := rec.acc
		if !acc.in_dirties {
			continue
		}
		times_dirtied, times_touched := acc.times_dirtied, acc.times_touched
		acc.times_dirtied, acc.times_touched, acc.in_dirties = 0, 0, false
		if times_dirtied == 0 {
			continue
		}
		addr := rec.addr
		if acc.suicided || delete_empty_accounts && is_empty(acc) {
			out.OnAccountDeleted(addr)
			self.accounts[addr] = nil
			continue
		}
		if times_dirtied == times_touched {
			continue
		}
		out.OnAccountChanged(addr, acc.AccountChange)
		acc.CodeDirty = false
		if len(acc.StorageDirty) == 0 {
			continue
		}
		if acc.storage_origin == nil {
			acc.storage_origin = make(AccountStorage, util.CeilPow2(len(acc.StorageDirty)))
		}
		for k, v := range acc.StorageDirty {
			acc.storage_origin[k] = v
		}
		acc.StorageDirty = nil
	}
}
