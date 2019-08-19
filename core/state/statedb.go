// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
	"sort"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/log"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/trie"
)

type revision struct {
	id           int
	journalIndex int
}
type Dirties = map[common.Address]struct{}
type Objects = map[common.Address]*stateObject
type Logs = map[common.Hash][]*types.Log
type StateChange Objects

var (
	// emptyState is the known hash of an empty state trie entry.
	emptyState = crypto.Keccak256Hash(nil)
	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256Hash(nil)
)

const NO_TRANSACTION = -1

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db   Database
	trie Trie
	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects      Objects
	stateObjectsDirty Dirties
	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// The refund counter, also used by state transitioning.
	refund uint64

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint

	preimages map[common.Hash][]byte

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int
}

// Create a new state from a given trie.
func New(root common.Hash, db Database) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		db:                db,
		trie:              tr,
		stateObjects:      make(Objects),
		stateObjectsDirty: make(Dirties),
		logs:              make(Logs),
		preimages:         make(map[common.Hash][]byte),
		journal:           newJournal(),
		txIndex:           NO_TRANSACTION,
	}, nil
}

func (this *StateDB) CommitStateChange(deleteEmptyObjects bool) StateChange {
	defer this.CheckPoint(true)
	ret := make(StateChange)
	for k := range this.journal.dirties {
		if _, exists := this.stateObjects[k]; exists {
			this.stateObjectsDirty[k] = struct{}{}
		}
	}
	for k := range this.stateObjectsDirty {
		stateObject := this.stateObjects[k]
		if stateObject.suicided || (deleteEmptyObjects && stateObject.empty()) {
			stateObject.deleted = true
		}
		ret[k] = unlinkedCopy(stateObject)
		for k, v := range stateObject.dirtyStorage {
			stateObject.originStorage[k] = v
		}
		stateObject.dirtyStorage = make(Storage)
		if stateObject.code != nil && stateObject.dirtyCode {
			stateObject.dirtyCode = false
		}
	}
	return ret
}

func (this *StateDB) Reset() {
	this.stateObjects = make(map[common.Address]*stateObject)
	this.stateObjectsDirty = make(map[common.Address]struct{})
	this.thash = common.Hash{}
	this.bhash = common.Hash{}
	this.txIndex = 0
	this.logs = make(map[common.Hash][]*types.Log)
	this.logSize = 0
	this.preimages = make(map[common.Hash][]byte)
	this.clearJournalAndRefund()
}

func (this *StateDB) Merge(change StateChange) {
	for addr, thatObj := range change {
		this.stateObjectsDirty[addr] = struct{}{}
		thisObj := this.stateObjects[addr]
		if thisObj == nil {
			cpy := unlinkedCopy(thatObj)
			cpy.db = this
			this.stateObjects[addr] = cpy
			continue
		}
		if thatObj.dirtyCode {
			thisObj.dirtyCode = thatObj.dirtyCode
			thisObj.code = thatObj.code
			thisObj.data.CodeHash = thatObj.data.CodeHash
		}
		if thatObj.suicided {
			thisObj.suicided = thatObj.suicided
		}
		if thatObj.deleted {
			thisObj.deleted = thatObj.suicided
		}
		if balanceDelta := thatObj.BalanceDelta(); balanceDelta.Sign() != 0 {
			if thatObj.balanceAdditive {
				thisObj.data.Balance = util.Sum(thisObj.data.Balance, balanceDelta)
			} else {
				thisObj.data.Balance = new(big.Int).Set(thatObj.data.Balance)
			}
		}
		if nonceDelta := thatObj.NonceDelta(); nonceDelta != 0 {
			util.Assert(nonceDelta > 0)
			if thatObj.nonceAdditive {
				thisObj.data.Nonce += nonceDelta
			} else {
				thisObj.data.Nonce = thatObj.data.Nonce
			}
		}
		for k, v := range thatObj.dirtyStorage {
			if _, hasBeen := thisObj.originStorage[k]; !hasBeen {
				thisObj.originStorage[k] = thatObj.originStorage[k]
			}
			thisObj.dirtyStorage[k] = v
		}
		//thisObj.originStorage = thatObj.originStorage.Copy()
		//thisObj.dirtyStorage = thatObj.dirtyStorage.Copy()
		//mergeDirtyStorage(thisObj, thatObj)
	}
}

func mergeDirtyStorage(left, right *stateObject) {
	for k, dirtyValue := range right.dirtyStorage {
		originValue, present := right.originStorage[k]
		util.Assert(present)
		if originValue != dirtyValue {
			left.originStorage[k] = originValue
			left.dirtyStorage[k] = dirtyValue
		}
	}
}

func unlinkedCopy(object *stateObject) *stateObject {
	copy := newObject(nil, object.address, object.data)

	copy.originData = object.originData
	copy.balanceAdditive = object.balanceAdditive
	copy.nonceAdditive = object.nonceAdditive

	copy.data.Balance = new(big.Int).Set(object.data.Balance)
	copy.code = object.code
	mergeDirtyStorage(copy, object)
	copy.suicided = object.suicided
	copy.dirtyCode = object.dirtyCode
	copy.deleted = object.deleted
	return copy
}

// setError remembers the first non-nil error it is called with.
func (self *StateDB) setError(err error) {
	if self.dbErr == nil {
		self.dbErr = err
	}
}

func (self *StateDB) Error() error {
	return self.dbErr
}

func (self *StateDB) AddLog(log *types.Log) {
	util.Assert(self.txIndex != NO_TRANSACTION)
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

// AddPreimage records a SHA3 preimage seen by the VM.
func (self *StateDB) AddPreimage(hash common.Hash, preimage []byte) {
	if _, ok := self.preimages[hash]; !ok {
		self.journal.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		self.preimages[hash] = pi
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (self *StateDB) Preimages() map[common.Hash][]byte {
	return self.preimages
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

func (this *StateDB) HasBalance(address common.Address, amount *big.Int) bool {
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
	size, err := self.db.ContractCodeSize(stateObject.addrHash, common.BytesToHash(stateObject.CodeHash()))
	if err != nil {
		self.setError(err)
	}
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
func (self *StateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(self.db, hash)
	}
	return common.Hash{}
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (self *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(self.db, hash)
	}
	return common.Hash{}
}

// Database retrieves the low level database supporting the lower level trie ops.
func (self *StateDB) Database() Database {
	return self.db
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
	if stateObject != nil {
		stateObject.AddBalance(amount)
	}
}

// SubBalance subtracts amount from the account associated with addr.
func (self *StateDB) SubBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SubBalance(common.Big0)
		// TODO don't forget !!!
		//stateObject.SubBalance(amount)
	}
}

func (self *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetBalance(amount)
		stateObject.balanceAdditive = false
	}
}

func (self *StateDB) setNonce(addr common.Address, nonce uint64) *stateObject {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
	return stateObject
}

func (self *StateDB) SetNonce(addr common.Address, nonce uint64) {
	if obj := self.setNonce(addr, nonce); obj != nil {
		obj.nonceAdditive = false
	}
}

func (this *StateDB) AddNonce(addr common.Address, value uint64) {
	this.setNonce(addr, this.GetNonce(addr)+value)
}

func (self *StateDB) SetCode(addr common.Address, code []byte) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(crypto.Keccak256Hash(code), code)
	}
}

func (self *StateDB) SetState(addr common.Address, key, value common.Hash) {
	stateObject := self.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(self.db, key, value)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (self *StateDB) Suicide(addr common.Address) bool {
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return false
	}
	self.journal.append(suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)

	return true
}

//
// Setting, updating & deleting state object methods.
//

// updateStateObject writes the given object to the trie.
func (self *StateDB) updateStateObject(stateObject *stateObject) {
	addr := stateObject.Address()
	data, err := rlp.EncodeToBytes(stateObject)
	if err != nil {
		panic(fmt.Errorf("can't encode object at %x: %v", addr[:], err))
	}
	self.setError(self.trie.TryUpdate(addr[:], data))
}

// deleteStateObject removes the given object from the state trie.
func (self *StateDB) deleteStateObject(stateObject *stateObject) {
	stateObject.deleted = true
	addr := stateObject.Address()
	self.setError(self.trie.TryDelete(addr[:]))
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
	enc, err := self.trie.TryGet(addr[:])
	if len(enc) == 0 {
		self.setError(err)
		return nil
	}
	var data Account
	if err := rlp.DecodeBytes(enc, &data); err != nil {
		log.Error("Failed to decode state object", "addr", addr, "err", err)
		return nil
	}
	// Insert into the live set.
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
		stateObject, _ = self.createObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (self *StateDB) createObject(addr common.Address) (newobj, prev *stateObject) {
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
	newObj, prev := self.createObject(addr)
	if prev != nil {
		newObj.setBalance(prev.data.Balance)
	}
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (self *StateDB) Copy() *StateDB {
	// Copy all the basic fields, initialize the memory ones
	state := &StateDB{
		db:                self.db,
		trie:              self.db.CopyTrie(self.trie),
		stateObjects:      make(map[common.Address]*stateObject, len(self.journal.dirties)),
		stateObjectsDirty: make(map[common.Address]struct{}, len(self.journal.dirties)),
		refund:            self.refund,
		logs:              make(map[common.Hash][]*types.Log, len(self.logs)),
		logSize:           self.logSize,
		preimages:         make(map[common.Hash][]byte, len(self.preimages)),
		journal:           newJournal(),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range self.journal.dirties {
		// As documented [here](https://github.com/ethereum/go-ethereum/pull/16485#issuecomment-380438527),
		// and in the Finalise-method, there is a case where an object is in the journal but not
		// in the stateObjects: OOG after touch on ripeMD prior to Byzantium. Thus, we need to check for
		// nil
		if object, exist := self.stateObjects[addr]; exist {
			state.stateObjects[addr] = object.deepCopy(state)
			state.stateObjectsDirty[addr] = struct{}{}
		}
	}
	// Above, we don't copy the actual journal. This means that if the copy is copied, the
	// loop above will be a no-op, since the copy's journal is empty.
	// Thus, here we iterate over stateObjects, to enable copies of copies
	for addr := range self.stateObjectsDirty {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = self.stateObjects[addr].deepCopy(state)
			state.stateObjectsDirty[addr] = struct{}{}
		}
	}
	for hash, logs := range self.logs {
		cpy := make([]*types.Log, len(logs))
		for i, l := range logs {
			cpy[i] = new(types.Log)
			*cpy[i] = *l
		}
		state.logs[hash] = cpy
	}
	for hash, preimage := range self.preimages {
		state.preimages[hash] = preimage
	}
	return state
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

// Finalise finalises the state by removing the self destructed objects
// and clears the journal as well as the refunds.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	defer s.CheckPoint(false)
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

		if stateObject.suicided || (deleteEmptyObjects && stateObject.empty()) {
			s.deleteStateObject(stateObject)
		} else {
			stateObject.updateRoot(s.db)
			s.updateStateObject(stateObject)
		}
		s.stateObjectsDirty[addr] = struct{}{}
	}
}

// IntermediateRoot computes the current root hash of the state trie.
// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.Finalise(deleteEmptyObjects)
	return s.trie.Hash()
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (self *StateDB) BeginTransaction(thash, bhash common.Hash, ti int) {
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

func (this *StateDB) CheckPoint(resetDirties bool) {
	if resetDirties {
		for k := range this.stateObjectsDirty {
			this.stateObjects[k].reset()
			delete(this.stateObjectsDirty, k)
		}
	}
	this.clearJournalAndRefund()
	this.thash = common.Hash{}
	this.bhash = common.Hash{}
	this.txIndex = NO_TRANSACTION
}

func (s *StateDB) clearJournalAndRefund() {
	s.journal = newJournal()
	s.validRevisions = s.validRevisions[:0]
	s.refund = 0
}

// Commit writes the state to the underlying in-memory trie database.
func (s *StateDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) {
	defer s.CheckPoint(true)
	for addr := range s.journal.dirties {
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Commit objects to the trie.
	for addr := range s.stateObjectsDirty {
		obj := s.stateObjects[addr]
		if obj.suicided || deleteEmptyObjects && obj.empty() {
			// If the object has been removed, don't bother syncing it
			// and just mark it for deletion in the trie.
			s.deleteStateObject(obj)
		} else {
			// Write any contract code associated with the state object
			if obj.code != nil && obj.dirtyCode {
				s.db.TrieDB().InsertBlob(common.BytesToHash(obj.CodeHash()), obj.code)
				obj.dirtyCode = false
			}
			// Write any storage changes in the state object to its storage trie.
			if err := obj.CommitTrie(s.db); err != nil {
				return common.Hash{}, err
			}
			// Update the object in the main account trie.
			s.updateStateObject(obj)
		}
	}
	root, err = s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyState {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			s.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}
