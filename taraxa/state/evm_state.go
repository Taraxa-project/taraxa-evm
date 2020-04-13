package state

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/dbg"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"math/big"
	"sort"
	"strings"
)

type EVMState struct {
	in       EVMStateInput
	refund   uint64
	logs     []vm.LogRecord
	accounts map[common.Address]*local_account
	dirties  map[common.Address]*local_account
	reverts  []func()
}
type local_account = struct {
	AccountChange
	storage_origin AccountStorage
	suicided       bool
	times_touched  int
	times_dirtied  int
}
type EVMStateInput interface {
	GetCode(code_hash *common.Hash) []byte
	GetAccount(addr *common.Address) (Account, bool)
	GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int
}
type AccountChange = struct {
	Account
	code          []byte
	code_dirty    bool
	storage_dirty AccountStorage
}
type AccountStorage = map[common.Hash]*big.Int
type EvmStateOpts = struct {
	AccountCacheSize      int
	DirtyAccountCacheSize int
}

func NewEVMState(src EVMStateInput, opts EvmStateOpts) (ret EVMState) {
	ret.in = src
	ret.accounts = make(map[common.Address]*local_account, opts.AccountCacheSize)
	ret.dirties = make(map[common.Address]*local_account, opts.DirtyAccountCacheSize)
	ret.reverts = make([]func(), 0, opts.DirtyAccountCacheSize*3)
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
	return amount.Sign() == 0 || self.GetBalance(address).Cmp(amount) >= 0
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

var ripemd_addr = common.BytesToAddress([]byte{3})

func (self *EVMState) AddBalance(addr common.Address, amount *big.Int) {
	acc := self.get_or_create_account(addr)
	if amount.Sign() != 0 {
		self.set_balance(addr, acc, new(big.Int).Add(acc.balance, amount))
		return
	}
	if !acc.is_empty() {
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
		self.set_balance(addr, acc, new(big.Int).Sub(acc.balance, amount))
	}
}

func (self *EVMState) set_balance(addr common.Address, acc *local_account, amount *big.Int) {
	balance_prev := acc.balance
	self.add_acc_revert(addr, acc, func() {
		acc.balance = balance_prev
	})
	acc.balance = amount
}

func (self *EVMState) IncrementNonce(addr common.Address) {
	acc := self.get_or_create_account(addr)
	self.add_acc_revert(addr, acc, func() {
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
	self.add_acc_revert(addr, acc, func() {
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
	self.add_acc_revert(addr, acc, func() {
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
	self.add_acc_revert(addr, acc, func() {
		acc.suicided, acc.balance = suicided_prev, balance_prev
	})
	acc.suicided, acc.balance = true, common.Big0
}

func (self *EVMState) get_or_create_account(addr common.Address) *local_account {
	if acc := self.get_account(addr); acc != nil {
		return acc
	}
	new := new(local_account)
	new.balance = common.Big0
	self.add_acc_revert(addr, new, func() {
		self.accounts[addr] = nil
		delete(self.dirties, addr)
	})
	self.accounts[addr] = new
	return new
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
	if acc.storage_root_hash == nil {
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
	self.dirties[addr] = acc
	acc.times_dirtied++
	self.add_revert(func() {
		acc.times_dirtied--
		revert()
	})
}

func (self *EVMState) add_revert(revert func()) {
	self.reverts = append(self.reverts, revert)
}

type EVMStateOutput = struct {
	OnAccountChanged func(address common.Address, change AccountChange)
	OnAccountDeleted func(address common.Address)
}

func (self *EVMState) Commit(delete_empty_accounts bool, sink EVMStateOutput) {
	self.reverts, self.logs, self.refund = self.reverts[:0], self.logs[:0], 0
	type dbg_rec = struct {
		key string
		val string
	}
	var recs []dbg_rec
	for addr, acc := range self.dirties {
		delete(self.dirties, addr)
		times_dirtied, times_touched := acc.times_dirtied, acc.times_touched
		acc.times_dirtied, acc.times_touched = 0, 0
		if times_dirtied == 0 {
			continue
		}
		if acc.suicided || delete_empty_accounts && acc.is_empty() {
			sink.OnAccountDeleted(addr)
			self.accounts[addr] = nil
			if dbg.Debugging && dbg.DebugStateCommit {
				recs = append(recs, dbg_rec{addr.Hex(), "   DELETED"})
			}
			continue
		}
		if times_dirtied == times_touched {
			continue
		}
		if dbg.Debugging && dbg.DebugStateCommit {
			recs = append(recs, dbg_rec{
				addr.Hex(),
				fmt.Sprint(
					"   nonce: ", acc.nonce,
					"\n   balance: ", acc.balance.String(),
					"\n   code_hash: ", func() string {
						if acc.code_hash == nil {
							return "NIL"
						}
						return acc.code_hash.Hex()
					}(),
					"\n   storage: ", func() string {
						var recs []dbg_rec
						for k, v := range acc.storage_dirty {
							if v.Sign() == 0 {
								recs = append(recs, dbg_rec{k.Hex(), "DELETED"})
							} else {
								recs = append(recs, dbg_rec{k.Hex(), common.BigToHash(v).Hex()})
							}
						}
						sort.Slice(recs, func(i, j int) bool {
							return strings.Compare(recs[i].key, recs[j].key) < 0
						})
						ret_str := ""
						for _, rec := range recs {
							ret_str += "    \n" + rec.key + " :: " + rec.val
						}
						return ret_str
					}(),
				),
			})
		}
		sink.OnAccountChanged(addr, acc.AccountChange)
		acc.code_dirty = false
		if len(acc.storage_dirty) == 0 {
			continue
		}
		if acc.storage_origin == nil {
			acc.storage_origin = make(AccountStorage, util.CeilPow2(len(acc.storage_dirty)))
		}
		for k, v := range acc.storage_dirty {
			acc.storage_origin[k] = v
		}
		acc.storage_dirty = nil
	}
	if dbg.Debugging && dbg.DebugStateCommit {
		sort.Slice(recs, func(i, j int) bool {
			return strings.Compare(recs[i].key, recs[j].key) < 0
		})
		for _, rec := range recs {
			fmt.Println("ACC CHANGE:", rec.key)
			fmt.Println(rec.val)
		}
	}
}
