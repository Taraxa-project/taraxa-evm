package state_evm

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type Account struct {
	AccountMapEntryHeader
	EVMStateAccountHeader
	*AccountBody
}
type AccountBody struct {
	AccountChange
	suicided       bool
	sink           AccountMutation
	storage_origin EVMStorage
	times_touched  uint32
	mod_count      uint32
}

func (self *Account) Address() *common.Address {
	return &self.addr
}

// TODO invert
func (self *Account) IsNotNIL() bool {
	return self.AccountBody != nil
}

func (self *Account) set_NIL() {
	self.AccountBody = nil
}

func (self *Account) IsEIP161Empty() bool {
	return !self.IsNotNIL() || self.Nonce == 0 && self.Balance.Sign() == 0 && self.CodeSize == 0
}

func (self *Account) GetBalance() *big.Int {
	if !self.IsNotNIL() {
		return bigutil.Big0
	}
	return self.Balance
}

func (self *Account) GetNonce() uint64 {
	if !self.IsNotNIL() {
		return 0
	}
	return self.Nonce
}

func (self *Account) GetCodeHash() *common.Hash {
	if !self.IsNotNIL() {
		return nil
	}
	if self.CodeSize != 0 {
		return self.CodeHash
	}
	return &crypto.EmptyBytesKeccak256
}

func (self *Account) GetCode() []byte {
	if !self.IsNotNIL() {
		return nil
	}
	if self.CodeSize == 0 {
		return nil
	}
	if self.Code == nil {
		self.Code = self.host.in.GetCode(self.CodeHash)
	}
	return self.Code
}

func (self *Account) GetCodeSize() uint64 {
	if !self.IsNotNIL() {
		return 0
	}
	return self.CodeSize
}

func (self *Account) GetState(key *big.Int) (ret *big.Int) {
	if !self.IsNotNIL() {
		return bigutil.Big0
	}
	key_b := bigutil.UnsafeUnsignedBytes(key)
	if value, present := self.StorageDirty[bigutil.UnsignedStr(key_b)]; present {
		return value
	}
	return self.get_committed_state(key, key_b)
}

func (self *Account) GetCommittedState(key *big.Int) (ret *big.Int) {
	return self.get_committed_state(key, bigutil.UnsafeUnsignedBytes(key))
}

func (self *Account) SetState(key, value *big.Int) {
	self.ensure_exists()
	prev := self.GetState(key)
	if prev.Cmp(value) == 0 {
		return
	}
	key_str := bigutil.UnsignedStr(bigutil.UnsafeUnsignedBytes(key))
	self.register_change(func() {
		self.StorageDirty[key_str] = prev
	})
	if self.StorageDirty == nil {
		self.StorageDirty = make(EVMStorage)
	}
	self.StorageDirty[key_str] = new(big.Int).Set(value)
}

func (self *Account) get_committed_state(key *big.Int, key_b bigutil.UnsignedBytes) *big.Int {
	if !self.IsNotNIL() {
		return bigutil.Big0
	}
	if self.storage_origin == nil {
		if self.StorageRootHash == nil {
			return bigutil.Big0
		}
	} else if ret, present := self.storage_origin[bigutil.UnsignedStr(key_b)]; present {
		return ret
	}
	ret, key_h := bigutil.Big0, self.host.bigconv.ToHash(key)
	self.host.GetAccountStorageFromDB(self.Address(), key_h, func(bytes []byte) {
		ret = bigutil.FromBytes(bytes)
	})
	if self.storage_origin == nil {
		self.storage_origin = make(EVMStorage)
	}
	self.storage_origin[bigutil.UnsignedStr(key_b)] = ret
	return ret
}

func (self *Account) HasSuicided() bool {
	return self.IsNotNIL() && self.suicided
}

var ripemd_addr = common.BytesToAddress([]byte{3})

func (self *Account) AddBalance(amount *big.Int) {
	self.ensure_exists()
	if amount.Sign() != 0 {
		self.set_balance(new(big.Int).Add(self.Balance, amount))
		return
	}
	if !self.IsEIP161Empty() {
		return
	}
	self.times_touched++
	self.register_change(func() {
		self.times_touched--
	})
	if self.addr == ripemd_addr {
		self.mod_count++
	}
}

func (self *Account) SubBalance(amount *big.Int) {
	self.ensure_exists()
	if amount.Sign() != 0 {
		self.set_balance(new(big.Int).Sub(self.Balance, amount))
	}
}

func (self *Account) set_balance(amount *big.Int) {
	balance_prev := self.Balance
	self.Balance = amount
	self.register_change(func() {
		self.Balance = balance_prev
	})
}

func (self *Account) IncrementNonce() {
	self.ensure_exists()
	self.Nonce++
	self.register_change(func() {
		self.Nonce--
	})
}

func (self *Account) SetCode(code []byte) {
	self.ensure_exists()
	assert.Holds(self.CodeSize == 0)
	code_size := len(code)
	if code_size == 0 {
		return
	}
	self.register_change(func() {
		self.CodeDirty, self.CodeHash, self.CodeSize, self.Code = false, nil, 0, nil
	})
	self.CodeDirty, self.CodeHash, self.CodeSize, self.Code = true, keccak256.Hash(code), uint64(code_size), code
}

func (self *Account) SetStateRawIrreversibly(key *common.Hash, value []byte) {
	self.ensure_exists()
	if self.RawStorageDirty == nil {
		self.RawStorageDirty = make(RawStorage)
	}
	self.RawStorageDirty[*key] = value
}

func (self *Account) Suicide(newAddr *common.Address) {
	new_acc := self.host.GetAccountConcrete(newAddr)
	if !self.IsNotNIL() {
		new_acc.AddBalance(bigutil.Big0)
		return
	}
	new_acc.AddBalance(self.Balance)
	suicided, balance_prev := self.suicided, self.Balance
	self.register_change(func() {
		self.suicided, self.Balance = suicided, balance_prev
	})
	self.suicided, self.Balance = true, bigutil.Big0
}

func (self *Account) ensure_exists() {
	if self.IsNotNIL() {
		return
	}
	self.AccountBody = new(AccountBody)
	self.AccountBody.Balance = bigutil.Big0
	was_deleted := self.deleted
	self.deleted = false
	self.register_change(func() {
		self.set_NIL()
		self.deleted = was_deleted
	})
}

func (self *Account) register_change(revert func()) {
	self.mod_count++
	self.host.register_change(func() {
		self.mod_count--
		revert()
	})
}

type acc_dirty_status = byte

const (
	unmodified acc_dirty_status = iota
	updated
	deleted
)

func (self *Account) flush(out Output, eip158 bool) acc_dirty_status {
	if !self.IsNotNIL() {
		return unmodified
	}
	mod_count, times_touched := self.mod_count, self.times_touched
	self.mod_count, self.times_touched = 0, 0
	if mod_count == 0 {
		return unmodified
	}
	if self.suicided || eip158 && self.IsEIP161Empty() {
		out.Delete(&self.addr)
		self.set_NIL()
		return deleted
	}
	if mod_count == times_touched {
		return unmodified
	}
	if self.sink == nil {
		self.sink = out.StartMutation(self.Address())
	}
	self.sink.Update(self.AccountChange)
	self.CodeDirty = false
	self.RawStorageDirty = nil
	if len(self.StorageDirty) != 0 {
		if self.storage_origin == nil {
			self.storage_origin = make(EVMStorage, util.CeilPow2(len(self.StorageDirty)))
		}
		for k, v := range self.StorageDirty {
			self.storage_origin[k] = v
		}
		self.StorageDirty = nil
	}
	return updated
}

func (self *Account) unload() {
	self.host.accounts.Delete(self)
	self.set_NIL()
}
