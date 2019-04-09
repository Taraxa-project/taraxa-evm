package conflict_tracking

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

const balance = "balance"
const code = "code"
const nonce = "nonce"

type ConflictTrackingStateDB struct {
	txId             TxId
	stateDB          *state.StateDB
	conflictDetector *ConflictDetector
}

func (this *ConflictTrackingStateDB) Init(txId TxId, commonDB *state.StateDB, conflicts *ConflictDetector) *ConflictTrackingStateDB {
	this.txId = txId
	this.stateDB = commonDB
	this.conflictDetector = conflicts
	return this
}

func (this *ConflictTrackingStateDB) CreateAccount(addr common.Address) {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
	})
	this.stateDB.CreateAccount(addr)
}

func (this *ConflictTrackingStateDB) SubBalance(addr common.Address, value *big.Int) {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
		key:     balance,
	})
	this.stateDB.SubBalance(addr, value)
}

func (this *ConflictTrackingStateDB) AddBalance(addr common.Address, value *big.Int) {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
		key:     balance,
	})
	this.stateDB.AddBalance(addr, value)
}

func (this *ConflictTrackingStateDB) GetBalance(addr common.Address) *big.Int {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     balance,
	})
	return this.stateDB.GetBalance(addr)
}

func (this *ConflictTrackingStateDB) GetNonce(addr common.Address) uint64 {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     nonce,
	})
	return this.stateDB.GetNonce(addr)
}

func (this *ConflictTrackingStateDB) SetNonce(addr common.Address, value uint64) {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
		key:     nonce,
	})
	this.stateDB.SetNonce(addr, value)
}

func (this *ConflictTrackingStateDB) SetCode(addr common.Address, val []byte) {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
		key:     code,
	})
	this.stateDB.SetCode(addr, val)
}

func (this *ConflictTrackingStateDB) GetCodeHash(addr common.Address) common.Hash {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     code,
	})
	return this.stateDB.GetCodeHash(addr)
}

func (this *ConflictTrackingStateDB) GetCode(addr common.Address) []byte {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     code,
	})
	return this.stateDB.GetCode(addr)
}

func (this *ConflictTrackingStateDB) GetCodeSize(addr common.Address) int {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     code,
	})
	return this.stateDB.GetCodeSize(addr)
}

func (this *ConflictTrackingStateDB) Suicide(addr common.Address) bool {
	// TODO???
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
	})
	return this.stateDB.Suicide(addr)
}

func (this *ConflictTrackingStateDB) HasSuicided(addr common.Address) bool {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
	})
	return this.stateDB.HasSuicided(addr)
}

func (this *ConflictTrackingStateDB) Exist(addr common.Address) bool {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
	})
	return this.stateDB.Exist(addr)
}

func (this *ConflictTrackingStateDB) Empty(addr common.Address) bool {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
	})
	return this.stateDB.Empty(addr)
}

func (this *ConflictTrackingStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     hash.Hex(),
	})
	return this.stateDB.GetCommittedState(addr, hash)
}

func (this *ConflictTrackingStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		account: addr,
		key:     key.Hex(),
	})
	return this.stateDB.GetState(addr, key)
}

func (this *ConflictTrackingStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	this.conflictDetector.Submit(&operation{
		txId:    this.txId,
		isWrite: true,
		account: addr,
		key:     key.Hex(),
	})
	this.stateDB.SetState(addr, key, value)
}

func (this *ConflictTrackingStateDB) AddLog(log *types.Log) {
	util.Assert(TxId(log.TxIndex) == this.txId)
	this.stateDB.AddLog(log)
}

func (this *ConflictTrackingStateDB) AddRefund(val uint64) {
	this.stateDB.AddRefund(val)
}

func (this *ConflictTrackingStateDB) SubRefund(val uint64) {
	this.stateDB.SubRefund(val)
}

func (this *ConflictTrackingStateDB) GetRefund() uint64 {
	return this.stateDB.GetRefund()
}

func (this *ConflictTrackingStateDB) RevertToSnapshot(pos int) {
	// Do nothing, because this instance is not meant to be reused
}

func (this *ConflictTrackingStateDB) Snapshot() int {
	// This is not needed, but left for compatibility
	return this.stateDB.Snapshot()
}

func (this *ConflictTrackingStateDB) AddPreimage(hash common.Hash, val []byte) {
	this.stateDB.AddPreimage(hash, val)
}
