package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"math/big"
)

type state_object struct {
	address           common.Address
	nonce             uint64
	balance           *big.Int
	storage_root_hash []byte
	code              code
	originStorage     storage
	dirtyStorage      storage
	suicided          bool
	deleted           bool
	times_touched     int
	db                *StateDB
	trie              *trie.Trie
}
type storage = map[common.Hash]common.Hash
type code = struct {
	hash  []byte
	size  uint64
	val   []byte
	dirty bool
}

func new_object(db *StateDB, address common.Address) *state_object {
	return &state_object{
		address:       address,
		balance:       common.Big0,
		originStorage: make(storage),
		dirtyStorage:  make(storage),
		db:            db,
	}
}

func (self *state_object) empty() bool {
	return self.nonce == 0 && self.balance.Sign() == 0 && self.code.size == 0
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (self *state_object) touch() {
	self.times_touched++
	self.db.journal.append(touchChange{
		account: self,
	})
	if self.address == ripemd {
		self.db.journal.dirty(self.address)
	}
}

func (self *state_object) get_or_open_trie() *trie.Trie {
	if self.trie == nil {
		self.trie = self.db.db.OpenStorageTrie(self.storage_root_hash, self.address)
	}
	return self.trie
}

func (self *state_object) get_state(key common.Hash) common.Hash {
	if value, ok := self.dirtyStorage[key]; ok {
		return value
	}
	return self.get_committed_state(key)
}

func (self *state_object) get_committed_state(key common.Hash) (ret common.Hash) {
	if val, ok := self.originStorage[key]; ok {
		return val
	}
	if len(self.storage_root_hash) == 0 {
		return
	}
	enc := self.get_or_open_trie().Get(key[:])
	if len(enc) != 0 {
		_, content, _, err := rlp.Split(enc)
		util.PanicIfNotNil(err)
		ret.SetBytes(content)
	}
	self.originStorage[key] = ret
	return
}

func (self *state_object) set_state(key, value common.Hash) {
	prev := self.get_state(key)
	if prev == value {
		return
	}
	self.db.journal.append(storageChange{
		account:  &self.address,
		key:      key,
		prevalue: prev,
	})
	self.dirtyStorage[key] = value
}

func (self *state_object) add_balance(amount *big.Int) {
	if amount.Sign() == 0 {
		if self.empty() {
			self.touch()
		}
		return
	}
	self.set_balance(new(big.Int).Add(self.balance, amount))
}

func (self *state_object) sub_balance(amount *big.Int) {
	if amount.Sign() != 0 {
		self.set_balance(new(big.Int).Sub(self.balance, amount))
	}
}

func (self *state_object) set_balance(amount *big.Int) {
	self.db.journal.append(balanceChange{
		account: &self.address,
		prev:    new(big.Int).Set(self.balance),
	})
	self.balance = amount
}

func (self *state_object) get_code() []byte {
	if self.code.size == 0 {
		return nil
	}
	if len(self.code.val) != 0 {
		return self.code.val
	}
	self.code.val = self.db.db.GetCommitted(self.code.hash)
	return self.code.val
}

func (self *state_object) set_code(val []byte) {
	self.db.journal.append(codeChange{
		account:  &self.address,
		prevcode: self.code,
	})
	if len(val) != 0 {
		self.code = code{util.Keccak256Pooled(val), uint64(len(val)), val, true}
	} else {
		self.code = code{nil, 0, nil, true}
	}
}

func (self *state_object) set_nonce(nonce uint64) {
	self.db.journal.append(nonceChange{
		account: &self.address,
		prev:    self.nonce,
	})
	self.nonce = nonce
}
