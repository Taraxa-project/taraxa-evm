// Copyright 2015 The go-ethereum Authors
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

package conflict_tracking

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"math/big"
)

type ConflictTrackingStateDB struct {
	txId      TxId
	commonDB  *state.StateDB
	conflicts *Conflicts
}

func (this *ConflictTrackingStateDB) Init(txId TxId, commonDB *state.StateDB, conflicts *Conflicts) *ConflictTrackingStateDB {
	this.txId = txId
	this.commonDB = commonDB
	this.conflicts = conflicts
	return this
}

func (this *ConflictTrackingStateDB) CreateAccount(addr common.Address) {
	//this.conflicts.getAccount(addr).writes[this.txId] = dummy
	this.commonDB.CreateAccount(addr)
}

func (this *ConflictTrackingStateDB) SubBalance(addr common.Address, value *big.Int) {
	// TODO
	this.commonDB.SubBalance(addr, value)
}

func (this *ConflictTrackingStateDB) AddBalance(addr common.Address, value *big.Int) {
	// TODO
	this.commonDB.AddBalance(addr, value)
}

func (this *ConflictTrackingStateDB) GetBalance(addr common.Address) *big.Int {
	// TODO
	return this.commonDB.GetBalance(addr)
}

func (this *ConflictTrackingStateDB) GetNonce(addr common.Address) uint64 {
	// TODO
	return this.commonDB.GetNonce(addr)
}

func (this *ConflictTrackingStateDB) SetNonce(addr common.Address, value uint64) {
	// TODO
	this.commonDB.SetNonce(addr, value)
}

func (this *ConflictTrackingStateDB) GetCodeHash(addr common.Address) common.Hash {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.commonDB.GetCodeHash(addr)
}

func (this *ConflictTrackingStateDB) GetCode(addr common.Address) []byte {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.commonDB.GetCode(addr)
}

func (this *ConflictTrackingStateDB) SetCode(addr common.Address, val []byte) {
	//this.conflicts.getAccount(addr).writes[this.txId] = dummy
	this.commonDB.SetCode(addr, val)
}

func (this *ConflictTrackingStateDB) GetCodeSize(addr common.Address) int {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.commonDB.GetCodeSize(addr)
}

func (this *ConflictTrackingStateDB) AddRefund(val uint64) {
	// TODO
	this.commonDB.AddRefund(val)
}

func (this *ConflictTrackingStateDB) SubRefund(val uint64) {
	// TODO
	this.commonDB.SubRefund(val)
}

func (this *ConflictTrackingStateDB) GetRefund() uint64 {
	// TODO
	return this.commonDB.GetRefund()
}

func (this *ConflictTrackingStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	// TODO
	return this.commonDB.GetCommittedState(addr, hash)
}

func (this *ConflictTrackingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	val := this.commonDB.GetState(addr, key)
	cell := this.conflicts.getAccount(addr).GetMemory(key)
	cell.reads[this.txId] = val
	if len(cell.writes) > 0 {
		conflictFound := false
		for txId, _ := range cell.writes {
			if txId != this.txId {
				conflictFound = true
				this.conflicts.conflictingTransactions[txId] = dummy
			}
		}
		if conflictFound {
			this.conflicts.conflictingTransactions[this.txId] = dummy
		}
	}
	return val
}

func (this *ConflictTrackingStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	cell := this.conflicts.getAccount(addr).GetMemory(key)
	cell.writes[this.txId] = value
	if len(cell.reads) > 0 {
		conflictFound := false
		for txId, _ := range cell.reads {
			if txId != this.txId {
				conflictFound = true
				this.conflicts.conflictingTransactions[txId] = dummy
			}
		}
		if conflictFound {
			this.conflicts.conflictingTransactions[this.txId] = dummy
		}
	}
	this.commonDB.SetState(addr, key, value)
}

func (this *ConflictTrackingStateDB) Suicide(addr common.Address) bool {
	//this.conflicts.getAccount(addr).writes[this.txId] = dummy
	return this.commonDB.Suicide(addr)
}

func (this *ConflictTrackingStateDB) HasSuicided(addr common.Address) bool {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.commonDB.HasSuicided(addr)
}

func (this *ConflictTrackingStateDB) Exist(addr common.Address) bool {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.commonDB.Exist(addr)
}

func (this *ConflictTrackingStateDB) Empty(addr common.Address) bool {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.commonDB.Empty(addr)
}

func (this *ConflictTrackingStateDB) RevertToSnapshot(pos int) {
	panic("shouldn't be called")
	//this.commonDB.RevertToSnapshot(pos)
}

func (this *ConflictTrackingStateDB) Snapshot() int {
	// TODO this is potentially not needed
	return this.commonDB.Snapshot()
}

func (this *ConflictTrackingStateDB) AddLog(log *types.Log) {
	panic("not supported yet")
	//this.commonDB.AddLog(log)
}

func (this *ConflictTrackingStateDB) AddPreimage(hash common.Hash, val []byte) {
	panic("not supported yet")
	//this.commonDB.AddPreimage(hash, val)
}
