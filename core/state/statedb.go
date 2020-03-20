package state

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"math/big"
	"sort"
)

// TODO reinstate async operations
type StateDB struct {
	db                *Database
	trie              *trie.Trie
	stateObjects      state_objects
	stateObjectsDirty state_objects
	refund            uint64
	tx_hash           common.Hash
	tx_index          int
	logs              map[common.Hash][]*types.Log
	logSize           uint
	journal           *journal
	validRevisions    []revision
	nextRevisionId    int
	encoder           *rlp.Encoder
}
type state_objects = map[common.Address]*state_object
type revision struct {
	id           int
	journalIndex int
}

func New(root common.Hash, db *Database) *StateDB {
	var trie_root []byte
	if root != empty_rlp_list_hash && root != common.ZeroHash {
		trie_root = root[:]
	}
	return &StateDB{
		db:                db,
		trie:              db.OpenTrie(trie_root),
		stateObjects:      make(state_objects),
		stateObjectsDirty: make(state_objects),
		logs:              make(map[common.Hash][]*types.Log),
		journal:           newJournal(),
		encoder:           rlp.NewEncoder(rlp.EncoderConfig{1 << 8, 1}),
	}
}

func (self *StateDB) Exist(addr common.Address) bool {
	return self.getStateObject(addr) != nil
}

func (self *StateDB) Empty(addr common.Address) bool {
	o := self.getStateObject(addr)
	return o == nil || o.empty()
}

func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	if o := self.getStateObject(addr); o != nil {
		return o.balance
	}
	return common.Big0
}

func (self *StateDB) BalanceEQ(address common.Address, amount *big.Int) bool {
	return self.GetBalance(address).Cmp(amount) == 0
}

func (self *StateDB) AssertBalanceGTE(address common.Address, amount *big.Int) bool {
	return self.GetBalance(address).Cmp(amount) >= 0
}

func (self *StateDB) GetNonce(addr common.Address) uint64 {
	if o := self.getStateObject(addr); o != nil {
		return o.nonce
	}
	return 0
}

func (self *StateDB) GetCode(addr common.Address) []byte {
	if o := self.getStateObject(addr); o != nil {
		return o.get_code()
	}
	return nil
}

func (self *StateDB) GetCodeSize(addr common.Address) uint64 {
	if o := self.getStateObject(addr); o != nil {
		return o.code.size
	}
	return 0
}

func (self *StateDB) GetCodeHash(addr common.Address) common.Hash {
	if o := self.getStateObject(addr); o != nil {
		if o.code.size == 0 {
			return crypto.EmptyBytesKeccak256
		}
		return common.BytesToHash(o.code.hash)
	}
	return common.Hash{}
}

func (self *StateDB) GetState(addr common.Address, hash common.Hash) (ret common.Hash) {
	if o := self.getStateObject(addr); o != nil {
		return o.get_state(hash)
	}
	return common.Hash{}
}

func (self *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	if o := self.getStateObject(addr); o != nil {
		return o.get_committed_state(hash)
	}
	return common.Hash{}
}

func (self *StateDB) HasSuicided(addr common.Address) bool {
	if o := self.getStateObject(addr); o != nil {
		return o.suicided
	}
	return false
}

func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	self.GetOrNewStateObject(addr).add_balance(amount)
}

func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	self.GetOrNewStateObject(addr).sub_balance(amount)
}

func (self *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	self.GetOrNewStateObject(addr).set_balance(amount)
}

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	self.GetOrNewStateObject(addr).set_nonce(nonce)
}

func (self *StateDB) IncrementNonce(addr common.Address) {
	self.SetNonce(addr, self.GetNonce(addr)+1)
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	self.GetOrNewStateObject(addr).set_code(code)
}

func (self *StateDB) SetState(addr common.Address, key, value common.Hash) {
	self.GetOrNewStateObject(addr).set_state(key, value)
}

func (self *StateDB) Suicide(addr common.Address, newAddr common.Address) {
	o := self.getStateObject(addr)
	if o == nil {
		self.AddBalance(newAddr, common.Big0)
		return
	}
	self.AddBalance(newAddr, o.balance)
	self.journal.append(suicideChange{
		account:     &addr,
		prev:        o.suicided,
		prevbalance: new(big.Int).Set(o.balance),
	})
	o.suicided = true
	o.balance = new(big.Int)
}

func (self *StateDB) CreateAccount(addr common.Address) {
	if new_o, prev := self.createOrResetObject(addr); prev != nil {
		new_o.balance = prev.balance
	}
}

func (self *StateDB) setStateObject(o *state_object) *state_object {
	self.stateObjects[o.address] = o
	return o
}

func (self *StateDB) GetOrNewStateObject(addr common.Address) *state_object {
	stateObject := self.getStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = self.createOrResetObject(addr)
	}
	return stateObject
}

func (self *StateDB) createOrResetObject(addr common.Address) (newobj, prev *state_object) {
	prev = self.getStateObject(addr)
	newobj = new_object(self, addr)
	if prev == nil {
		self.journal.append(createObjectChange{account: &addr})
	} else {
		self.journal.append(resetObjectChange{prev: prev})
	}
	self.setStateObject(newobj)
	return newobj, prev
}

func (self *StateDB) AddLog(log *types.Log) {
	self.journal.append(addLogChange{txhash: self.tx_hash})
	log.TxHash = self.tx_hash
	log.TxIndex = uint(self.tx_index)
	log.Index = self.logSize
	self.logs[self.tx_hash] = append(self.logs[self.tx_hash], log)
	self.logSize++
}

func (self *StateDB) GetLogs(hash common.Hash) []*types.Log {
	return self.logs[hash]
}

func (self *StateDB) AddRefund(gas uint64) {
	self.journal.append(refundChange{prev: self.refund})
	self.refund += gas
}

func (self *StateDB) SubRefund(gas uint64) {
	self.journal.append(refundChange{prev: self.refund})
	if gas > self.refund {
		panic("Refund counter below zero")
	}
	self.refund -= gas
}

func (self *StateDB) GetRefund() uint64 {
	return self.refund
}

func (self *StateDB) Snapshot() int {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.length()})
	return id
}

func (self *StateDB) RevertToSnapshot(revid int) {
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex
	self.journal.revert(self, snapshot)
	self.validRevisions = self.validRevisions[:idx]
}

func (self *StateDB) SetTransactionMetadata(tx_hash common.Hash, tx_index int) {
	self.tx_hash, self.tx_index = tx_hash, tx_index
}

func (self *StateDB) Checkpoint(deleteEmptyObjects bool) {
	defer self.clearJournalAndRefund()
	for addr, times_marked_dirty := range self.journal.dirties {
		stateObject, exist := self.stateObjects[addr]
		if !exist {
			// TODO resolve this stupidity
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `self.journal.dirties` but not in `self.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}
		touched_only := times_marked_dirty == stateObject.times_touched
		stateObject.times_touched = 0
		if stateObject.suicided || deleteEmptyObjects && stateObject.empty() {
			stateObject.deleted = true
			self.stateObjectsDirty[addr] = stateObject
			continue
		}
		if touched_only {
			continue
		}
		self.stateObjectsDirty[addr] = stateObject
		if len(stateObject.dirtyStorage) == 0 {
			continue
		}
		trie := stateObject.get_or_open_trie()
		for key, value := range stateObject.dirtyStorage {
			stateObject.originStorage[key] = value
			var pos byte
			for pos < common.HashLength && value[pos] == 0 {
				pos++
			}
			if pos == common.HashLength {
				trie.Delete(key[:])
				continue
			}
			self.encoder.AppendString(value[pos:])
			enc := self.encoder.ToBytes(nil)
			self.encoder.Reset()
			trie.Put(key[:], enc, enc)
		}
		stateObject.dirtyStorage = make(storage)
	}
}

func (self *StateDB) clearJournalAndRefund() {
	self.journal = newJournal()
	self.validRevisions = self.validRevisions[:0]
	self.refund = 0
}

// refactor account enc/dec

func dec_uint64(b []byte) uint64 {
	switch len(b) {
	case 0:
		return 0
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(b[0])<<8 | uint64(b[1])
	case 3:
		return uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		return uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		return uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 |
			uint64(b[4])
	case 6:
		return uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 |
			uint64(b[4])<<8 | uint64(b[5])
	case 7:
		return uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 |
			uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	panic("impossible")
}

func (self *StateDB) getStateObject(addr common.Address) *state_object {
	if o := self.stateObjects[addr]; o != nil {
		if o.deleted {
			return nil
		}
		return o
	}
	enc_storage := self.trie.Get(addr[:])
	if len(enc_storage) == 0 {
		return nil
	}
	return self.setStateObject(dec(new_object(self, addr), enc_storage))
}

func (self *StateDB) Commit() common.Hash {
	for addr, o := range self.stateObjectsDirty {
		if o.deleted {
			self.trie.Delete(addr[:])
			continue
		}
		if o.code.dirty {
			self.db.PutAsync(o.code.hash, o.code.val)
			o.code.dirty = false
		}
		if o.trie != nil {
			o.storage_root_hash = o.trie.CommitNodes()
		}
		enc_storage, enc_hash := enc(self.encoder, o)
		self.trie.Put(addr[:], enc_storage, enc_hash)
	}
	self.stateObjectsDirty = make(state_objects)
	if root := self.trie.CommitNodes(); len(root) != 0 {
		return common.BytesToHash(root)
	}
	return empty_rlp_list_hash
}

var empty_rlp_list_hash = func() common.Hash {
	b, err := rlp.EncodeToBytes([]byte(nil))
	util.PanicIfNotNil(err)
	return crypto.Keccak256Hash(b)
}()
