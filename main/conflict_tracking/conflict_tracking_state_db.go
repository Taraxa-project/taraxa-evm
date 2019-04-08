package conflict_tracking

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

type ConflictTrackingStateDB struct {
	txId      TxId
	StateDB   *state.StateDB
	conflicts *ConflictDetector
}

func (this *ConflictTrackingStateDB) Init(txId TxId, commonDB *state.StateDB, conflicts *ConflictDetector) *ConflictTrackingStateDB {
	this.txId = txId
	this.StateDB = commonDB
	this.conflicts = conflicts
	return this
}

func (this *ConflictTrackingStateDB) CreateAccount(addr common.Address) {
	//this.conflicts.getAccount(addr).writes[this.txId] = dummy
	this.StateDB.CreateAccount(addr)
}

func (this *ConflictTrackingStateDB) SubBalance(addr common.Address, value *big.Int) {
	// TODO
	this.StateDB.SubBalance(addr, value)
}

func (this *ConflictTrackingStateDB) AddBalance(addr common.Address, value *big.Int) {
	// TODO
	this.StateDB.AddBalance(addr, value)
}

func (this *ConflictTrackingStateDB) GetBalance(addr common.Address) *big.Int {
	// TODO
	return this.StateDB.GetBalance(addr)
}

func (this *ConflictTrackingStateDB) GetNonce(addr common.Address) uint64 {
	// TODO
	return this.StateDB.GetNonce(addr)
}

func (this *ConflictTrackingStateDB) SetNonce(addr common.Address, value uint64) {
	// TODO
	this.StateDB.SetNonce(addr, value)
}

func (this *ConflictTrackingStateDB) GetCodeHash(addr common.Address) common.Hash {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.StateDB.GetCodeHash(addr)
}

func (this *ConflictTrackingStateDB) GetCode(addr common.Address) []byte {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.StateDB.GetCode(addr)
}

func (this *ConflictTrackingStateDB) SetCode(addr common.Address, val []byte) {
	//this.conflicts.getAccount(addr).writes[this.txId] = dummy
	this.StateDB.SetCode(addr, val)
}

func (this *ConflictTrackingStateDB) GetCodeSize(addr common.Address) int {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.StateDB.GetCodeSize(addr)
}

func (this *ConflictTrackingStateDB) AddRefund(val uint64) {
	// TODO
	this.StateDB.AddRefund(val)
}

func (this *ConflictTrackingStateDB) SubRefund(val uint64) {
	// TODO
	this.StateDB.SubRefund(val)
}

func (this *ConflictTrackingStateDB) GetRefund() uint64 {
	// TODO
	return this.StateDB.GetRefund()
}

func (this *ConflictTrackingStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	// TODO
	return this.StateDB.GetCommittedState(addr, hash)
}

func (this *ConflictTrackingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	val := this.StateDB.GetState(addr, key)
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
	this.StateDB.SetState(addr, key, value)
}

func (this *ConflictTrackingStateDB) Suicide(addr common.Address) bool {
	//this.conflicts.getAccount(addr).writes[this.txId] = dummy
	return this.StateDB.Suicide(addr)
}

func (this *ConflictTrackingStateDB) HasSuicided(addr common.Address) bool {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.StateDB.HasSuicided(addr)
}

func (this *ConflictTrackingStateDB) Exist(addr common.Address) bool {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.StateDB.Exist(addr)
}

func (this *ConflictTrackingStateDB) Empty(addr common.Address) bool {
	//this.conflicts.getAccount(addr).reads[this.txId] = dummy
	return this.StateDB.Empty(addr)
}

func (this *ConflictTrackingStateDB) RevertToSnapshot(pos int) {
	// Do nothing, because this instance is not meant to be reused
}

func (this *ConflictTrackingStateDB) Snapshot() int {
	// This is not needed, but left for compatibility
	return this.StateDB.Snapshot()
}

func (this *ConflictTrackingStateDB) AddLog(log *types.Log) {
	util.Assert(log.TxIndex == this.txId)
	this.StateDB.AddLog(log)
}

func (this *ConflictTrackingStateDB) AddPreimage(hash common.Hash, val []byte) {
	this.StateDB.AddPreimage(hash, val)
}
