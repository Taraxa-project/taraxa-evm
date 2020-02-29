package state

import (
	"bytes"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/log"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"math/big"
	"runtime"
	"sort"
	"sync/atomic"
)

type StateDB struct {
	db   *Database
	trie *trie.Trie
	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      StateObjects
	stateObjectsDirty StateObjects
	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error
	// The refund counter, also used by state transitioning.
	refund       uint64
	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint
	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int
}
type StateObjects = map[common.Address]*stateObject
type BalanceTable = map[common.Address]*hexutil.Big
type revision struct {
	id           int
	journalIndex int
}

// Create a new state from a given trie.
func New(root common.Hash, db *Database) *StateDB {
	return &StateDB{
		db:                db,
		trie:              db.OpenTrie(&root),
		stateObjects:      make(StateObjects),
		stateObjectsDirty: make(StateObjects),
		logs:              make(map[common.Hash][]*types.Log),
		journal:           newJournal(),
	}
}

// setError remembers the first non-nil error it is called with.
func (self *StateDB) setError(err error) {
	util.PanicIfNotNil(err)
	//TODO handle
	//if self.dbErr == nil {
	//	self.dbErr = err
	//}
}

func (self *StateDB) Error() error {
	return self.dbErr
}

func (self *StateDB) AddLog(log *types.Log) {
	self.journal.append(addLogChange{txhash: self.thash})

	log.TxHash = self.thash
	log.BlockHash = self.bhash
	log.TxIndex = uint(self.txIndex)
	log.Index = self.logSize
	self.logs[self.thash] = append(self.logs[self.thash], log)
	self.logSize++
}

func (self *StateDB) GetLogs(hash common.Hash) []*types.Log {
	return self.logs[hash]
}

func (self *StateDB) Logs() []*types.Log {
	var logs []*types.Log
	for _, lgs := range self.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

// AddRefund adds gas to the refund counter
func (self *StateDB) AddRefund(gas uint64) {
	self.journal.append(refundChange{prev: self.refund})
	self.refund += gas
}

// SubRefund removes gas from the refund counter.
// This method will panic if the refund counter goes below zero
func (self *StateDB) SubRefund(gas uint64) {
	self.journal.append(refundChange{prev: self.refund})
	if gas > self.refund {
		panic("Refund counter below zero")
	}
	self.refund -= gas
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (self *StateDB) Exist(addr common.Address) bool {
	return self.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (self *StateDB) Empty(addr common.Address) bool {
	so := self.getStateObject(addr)
	return so == nil || so.empty()
}

// Retrieve the balance from the given address or 0 if object not found
func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.Big0
}

func (this *StateDB) BalanceEQ(address common.Address, amount *big.Int) bool {
	return this.GetBalance(address).Cmp(amount) == 0
}

func (this *StateDB) AssertBalanceGTE(address common.Address, amount *big.Int) bool {
	return this.GetBalance(address).Cmp(amount) >= 0
}

func (self *StateDB) GetNonce(addr common.Address) uint64 {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}
	return 0
}

func (self *StateDB) GetCode(addr common.Address) []byte {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code(self.db)
	}
	return nil
}

func (self *StateDB) GetCodeSize(addr common.Address) int {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return 0
	}
	if stateObject.code != nil {
		return len(stateObject.code)
	}
	size, err := self.db.CodeSize(stateObject.CodeHash())
	self.setError(err)
	return size
}

func (self *StateDB) GetCodeHash(addr common.Address) common.Hash {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return common.Hash{}
	}
	return common.BytesToHash(stateObject.CodeHash())
}

// GetState retrieves a value from the given account's storage trie.
func (self *StateDB) GetState(addr common.Address, hash common.Hash) (ret common.Hash) {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(hash)
	}
	return common.Hash{}
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (self *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(hash)
	}
	return common.Hash{}
}

func (self *StateDB) HasSuicided(addr common.Address) bool {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

/*
 * SETTERS
 */

// AddBalance adds amount to the account associated with addr.
func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	stateObject.AddBalance(amount)
}

// SubBalance subtracts amount from the account associated with addr.
func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	stateObject.SubBalance(amount)
}

func (self *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	stateObject.SetBalance(amount)
}

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	stateObject := self.GetOrNewStateObject(addr)
	stateObject.SetNonce(nonce)
}

func (this *StateDB) IncrementNonce(addr common.Address) {
	this.SetNonce(addr, this.GetNonce(addr)+1)
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	hash, return_to_pool := util.Keccak256Pooled(code)
	stateObject.SetCode(common.BytesToHash(hash), code)
	go return_to_pool()
}

func (self *StateDB) SetState(addr common.Address, key, value common.Hash) {
	self.GetOrNewStateObject(addr).SetState(key, value)
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (self *StateDB) Suicide(addr common.Address, newAddr common.Address) {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		self.AddBalance(newAddr, common.Big0)
		return
	}
	self.AddBalance(newAddr, stateObject.Balance())
	self.journal.append(suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.setBalance(new(big.Int))
}

// Retrieve a state object given by the address. Returns nil if not found.
func (self *StateDB) getStateObject(addr common.Address) (stateObject *stateObject) {
	// Prefer 'live' objects.
	if obj := self.stateObjects[addr]; obj != nil {
		if obj.deleted {
			return nil
		}
		return obj
	}

	// Load the object from the database.
	enc, err := self.trie.Get(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	obj := newObject(self, addr, data)
	self.setStateObject(obj)
	return obj
}

func (self *StateDB) setStateObject(object *stateObject) {
	self.stateObjects[object.Address()] = object
}

// Retrieve a state object or create a new state object if nil.
func (self *StateDB) GetOrNewStateObject(addr common.Address) *stateObject {
	stateObject := self.getStateObject(addr)
	if stateObject == nil || stateObject.deleted {
		stateObject, _ = self.createOrResetObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (self *StateDB) createOrResetObject(addr common.Address) (newobj, prev *stateObject) {
	prev = self.getStateObject(addr)
	newobj = newObject(self, addr, Account{})
	newobj.setNonce(0) // sets the object to dirty
	if prev == nil {
		self.journal.append(createObjectChange{account: &addr})
	} else {
		self.journal.append(resetObjectChange{prev: prev})
	}
	self.setStateObject(newobj)
	return newobj, prev
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//   1. sends funds to sha(account ++ (nonce + 1))
//   2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
//
// Carrying over the balance ensures that Ether doesn't disappear.
func (self *StateDB) CreateAccount(addr common.Address) {
	newObj, prev := self.createOrResetObject(addr)
	if prev != nil {
		newObj.setBalance(prev.data.Balance)
	}
}

// Snapshot returns an identifier for the current revision of the state.
func (self *StateDB) Snapshot() int {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (self *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex
	// Replay the journal to undo changes and remove invalidated snapshots
	self.journal.revert(self, snapshot)
	self.validRevisions = self.validRevisions[:idx]
}

// GetRefund returns the current value of the refund counter.
func (self *StateDB) GetRefund() uint64 {
	return self.refund
}

func (self *StateDB) SetTransactionMetadata(thash, bhash common.Hash, ti int) {
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

func (s *StateDB) Checkpoint(deleteEmptyObjects bool) {
	defer s.clearJournalAndRefund()
	for addr := range s.journal.dirties {
		stateObject, exist := s.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}
		s.stateObjectsDirty[addr] = stateObject
		if stateObject.suicided || deleteEmptyObjects && stateObject.empty() {
			stateObject.deleted = true
			continue
		}
		if len(stateObject.dirtyStorage) == 0 {
			continue
		}
		tr := stateObject.getOrOpenTrie()
		for key, value := range stateObject.dirtyStorage {
			key := key
			stateObject.originStorage[key] = value
			if value == common.ZeroHash {
				tr.DeleteAsync(key[:])
				continue
			}
			v, err := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
			util.PanicIfNotNil(err)
			tr.InsertAsync(key[:], v)
		}
		stateObject.dirtyStorage = make(Storage)
	}
}

func (s *StateDB) clearJournalAndRefund() {
	s.journal = newJournal()
	s.validRevisions = s.validRevisions[:0]
	s.refund = 0
}

func (self *StateDB) Commit() (root common.Hash, err error) {
	child_tasks := int32(0)
	for addr, stateObject := range self.stateObjectsDirty {
		addr := addr
		if stateObject.deleted {
			self.trie.DeleteAsync(addr[:])
			continue
		}
		if stateObject.dirtyCode {
			self.db.PutAsync(stateObject.CodeHash(), stateObject.code)
			stateObject.dirtyCode = false
		}
		atomic.AddInt32(&child_tasks, 1)
		stateObject := stateObject
		go func() {
			defer atomic.AddInt32(&child_tasks, -1)
			root, err := stateObject.getOrOpenTrie().Commit()
			util.PanicIfNotNil(err)
			stateObject.data.Root = root
			enc, err := rlp.EncodeToBytes(stateObject)
			util.PanicIfNotNil(err)
			self.trie.InsertAsync(addr[:], enc)
		}()
	}
	self.stateObjectsDirty = make(StateObjects)
	for atomic.LoadInt32(&child_tasks) != 0 {
		runtime.Gosched()
	}
	defer log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return self.trie.Commit()
}
