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

// TODO "touch" https://github.com/ethereum/eips/issues/158
// TODO move to StateDB
type ConflictTrackingStateDB struct {
	txId             TxId
	stateDB          *state.StateDB
	conflictDetector *ConflictDetector
}

func (this *ConflictTrackingStateDB) Init(
	txId TxId, commonDB *state.StateDB, conflicts *ConflictDetector) *ConflictTrackingStateDB {
	this.txId = txId
	this.stateDB = commonDB
	this.conflictDetector = conflicts
	return this
}

func (this *ConflictTrackingStateDB) CreateAccount(addr common.Address) {
	this.onAccountWrite(addr)
	this.stateDB.CreateAccount(addr)
}

func (this *ConflictTrackingStateDB) SubBalance(addr common.Address, value *big.Int) {
	this.onAccountWrite(addr, balance)
	this.stateDB.SubBalance(addr, value)
}

func (this *ConflictTrackingStateDB) AddBalance(addr common.Address, value *big.Int) {
	this.onAccountWrite(addr, balance)
	this.stateDB.AddBalance(addr, value)
}

func (this *ConflictTrackingStateDB) GetBalance(addr common.Address) *big.Int {
	this.onAccountRead(addr, balance)
	return this.stateDB.GetBalance(addr)
}

func (this *ConflictTrackingStateDB) GetNonce(addr common.Address) uint64 {
	this.onAccountRead(addr, nonce)
	return this.stateDB.GetNonce(addr)
}

func (this *ConflictTrackingStateDB) SetNonce(addr common.Address, value uint64) {
	this.onAccountWrite(addr, nonce)
	this.stateDB.SetNonce(addr, value)
}

func (this *ConflictTrackingStateDB) SetCode(addr common.Address, val []byte) {
	this.onAccountWrite(addr, code)
	this.stateDB.SetCode(addr, val)
}

func (this *ConflictTrackingStateDB) GetCodeHash(addr common.Address) common.Hash {
	this.onAccountRead(addr, code)
	return this.stateDB.GetCodeHash(addr)
}

func (this *ConflictTrackingStateDB) GetCode(addr common.Address) []byte {
	this.onAccountRead(addr, code)
	return this.stateDB.GetCode(addr)
}

func (this *ConflictTrackingStateDB) GetCodeSize(addr common.Address) int {
	this.onAccountRead(addr, code)
	return this.stateDB.GetCodeSize(addr)
}

func (this *ConflictTrackingStateDB) Suicide(addr common.Address) bool {
	this.onAccountRead(addr)
	hasSuicided := this.stateDB.Suicide(addr)
	if hasSuicided {
		// read write
		this.onAccountWrite(addr, balance)
	}
	return hasSuicided
}

func (this *ConflictTrackingStateDB) HasSuicided(addr common.Address) bool {
	this.onAccountRead(addr)
	return this.stateDB.HasSuicided(addr)
}

func (this *ConflictTrackingStateDB) Exist(addr common.Address) bool {
	this.onAccountRead(addr)
	return this.stateDB.Exist(addr)
}

func (this *ConflictTrackingStateDB) Empty(addr common.Address) bool {
	this.onAccountRead(addr, nonce, balance, code)
	return this.stateDB.Empty(addr)
}

func (this *ConflictTrackingStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	this.onAccountRead(addr, hash.Hex())
	return this.stateDB.GetCommittedState(addr, hash)
}

func (this *ConflictTrackingStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	this.onAccountRead(addr, hash.Hex())
	return this.stateDB.GetState(addr, hash)
}

func (this *ConflictTrackingStateDB) SetState(addr common.Address, hash common.Hash, value common.Hash) {
	this.onAccountWrite(addr, hash.Hex())
	this.stateDB.SetState(addr, hash, value)
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

func (this *ConflictTrackingStateDB) onAccountRead(address common.Address, keys ...string) {
	accountKey := address.Hex()
	this.conflictDetector.Submit(&Operation{
		Author: this.txId,
		Key:    accountKey,
	})
	if len(keys) > 0 && this.stateDB.Exist(address) {
		for _, key := range keys {
			this.conflictDetector.Submit(&Operation{
				Author: this.txId,
				Key:    accountKey + key,
			})
		}
	}
}

func (this *ConflictTrackingStateDB) onAccountWrite(address common.Address, keys ...string) {
	accountKey := address.Hex()
	this.onAccountRead(address)
	if len(keys) == 0 || !this.stateDB.Exist(address) {
		this.conflictDetector.Submit(&Operation{
			IsWrite: true,
			Author:  this.txId,
			Key:     accountKey,
		})
	}
	for _, key := range keys {
		this.conflictDetector.Submit(&Operation{
			IsWrite: true,
			Author:  this.txId,
			Key:     accountKey + key,
		})
	}
}
