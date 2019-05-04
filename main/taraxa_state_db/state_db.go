package taraxa_state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_detector"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

const balance = "balance"
const code = "code"
const nonce = "nonce"

type TransientState struct {
	BalanceDeltas map[common.Address]*big.Int
	NonceDeltas   map[common.Address]uint64
}

type TaraxaStateDB struct {
	conflictLogger        conflict_detector.OperationLogger
	stateDB               *state.StateDB
	totalTransientState   *TransientState
	currentTransientState *TransientState
	lastCommittedSnapshot int
	transientStateLog     util.RevertLog
}

func New(stateDB *state.StateDB, conflictLogger conflict_detector.OperationLogger) *TaraxaStateDB {
	this := new(TaraxaStateDB)
	this.stateDB = stateDB
	this.totalTransientState = newTransientState()
	this.currentTransientState = newTransientState()
	this.conflictLogger = conflictLogger
	this.lastCommittedSnapshot = this.transientStateLog.CurrentSnapshot()
	return this
}

func (this *TaraxaStateDB) CreateAccount(addr common.Address) {
	preexisting := this.stateDB.Exist(addr)
	this.onCreateOrDeleteAccount(addr)
	if preexisting {
		this.conflictLogger(conflict_detector.SET, accountCompositeKey(addr, balance))
	}
	this.stateDB.CreateAccount(addr)
}

func (this *TaraxaStateDB) SubBalance(addr common.Address, value *big.Int) {
	this.addBalance(addr, new(big.Int).Neg(value))
	this.stateDB.SubBalance(addr, common.Big0)
}

func (this *TaraxaStateDB) AddBalance(addr common.Address, value *big.Int) {
	this.addBalance(addr, value)
	if value.Sign() == 0 {
		this.onAccountEmptyCheck(addr)
	}
	this.stateDB.AddBalance(addr, common.Big0)
}

func (this *TaraxaStateDB) addBalance(addr common.Address, value *big.Int) {
	this.onGetOrCreateAccount(addr)
	if value.Sign() == 0 {
		return
	}
	this.conflictLogger(conflict_detector.ADD, accountCompositeKey(addr, balance))
	this.modifyTransientState(func(s *TransientState) util.Revert {
		_, revert := util.Compute(s.BalanceDeltas, addr, func(_, oldVal interface{}, _ bool) interface{} {
			return util.Sum(oldVal.(*big.Int), value)
		})
		return revert
	})
}

func (this *TaraxaStateDB) GetBalance(addr common.Address) *big.Int {
	this.onAccountRead(addr, balance)
	return util.Sum(this.stateDB.GetBalance(addr), this.totalTransientState.BalanceDeltas[addr])
}

func (this *TaraxaStateDB) HasBalance(address common.Address, amount *big.Int) bool {
	return true
}

func (this *TaraxaStateDB) GetNonce(addr common.Address) uint64 {
	this.onAccountRead(addr, nonce)
	return this.stateDB.GetNonce(addr) + this.totalTransientState.NonceDeltas[addr]
}

// legacy
func (this *TaraxaStateDB) SetNonce(addr common.Address, value uint64) {
	panic("not expected to be called")
}

func (this *TaraxaStateDB) AddNonce(addr common.Address, val uint64) {
	this.onGetOrCreateAccount(addr)
	this.conflictLogger(conflict_detector.ADD, accountCompositeKey(addr, nonce))
	this.stateDB.AddNonce(addr, 0)
	this.modifyTransientState(func(s *TransientState) util.Revert {
		_, revert := util.Compute(s.NonceDeltas, addr, func(_, oldVal interface{}, _ bool) interface{} {
			return oldVal.(uint64) + val
		})
		return revert
	})
}

func (this *TaraxaStateDB) modifyTransientState(cb func(*TransientState) util.Revert) {
	this.transientStateLog.Append(cb(this.currentTransientState))
	this.transientStateLog.Append(cb(this.totalTransientState))
}

func (this *TaraxaStateDB) SetCode(addr common.Address, val []byte) {
	this.onAccountWrite(addr, code)
	this.stateDB.SetCode(addr, val)
}

func (this *TaraxaStateDB) GetCodeHash(addr common.Address) common.Hash {
	this.onAccountRead(addr, code)
	return this.stateDB.GetCodeHash(addr)
}

func (this *TaraxaStateDB) GetCode(addr common.Address) []byte {
	this.onAccountRead(addr, code)
	return this.stateDB.GetCode(addr)
}

func (this *TaraxaStateDB) GetCodeSize(addr common.Address) int {
	this.onAccountRead(addr, code)
	return this.stateDB.GetCodeSize(addr)
}

func (this *TaraxaStateDB) Suicide(addr common.Address) bool {
	this.onGetAccount(addr)
	hasSuicided := this.stateDB.Suicide(addr)
	if hasSuicided {
		this.onCreateOrDeleteAccount(addr)
	}
	return hasSuicided
}

func (this *TaraxaStateDB) HasSuicided(addr common.Address) bool {
	this.onGetAccount(addr)
	return this.stateDB.HasSuicided(addr)
}

func (this *TaraxaStateDB) Exist(addr common.Address) bool {
	this.onGetAccount(addr)
	return this.stateDB.Exist(addr)
}

func (this *TaraxaStateDB) Empty(addr common.Address) bool {
	this.onGetAccount(addr)
	if this.stateDB.Exist(addr) {
		this.onAccountEmptyCheck(addr)
	}
	return this.stateDB.Empty(addr)
}

func (this *TaraxaStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	this.onAccountRead(addr, hash.Hex())
	return this.stateDB.GetCommittedState(addr, hash)
}

func (this *TaraxaStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	this.onAccountRead(addr, hash.Hex())
	return this.stateDB.GetState(addr, hash)
}

func (this *TaraxaStateDB) SetState(addr common.Address, hash common.Hash, value common.Hash) {
	this.onAccountWrite(addr, hash.Hex())
	this.stateDB.SetState(addr, hash, value)
}

func (this *TaraxaStateDB) AddLog(log *types.Log) {
	// even though logs go into the state, they never produce conflicts because they
	// are scoped to the transaction id (hash)
	this.stateDB.AddLog(log)
}

func (this *TaraxaStateDB) AddRefund(val uint64) {
	this.stateDB.AddRefund(val)
}

func (this *TaraxaStateDB) SubRefund(val uint64) {
	this.stateDB.SubRefund(val)
}

func (this *TaraxaStateDB) GetRefund() uint64 {
	return this.stateDB.GetRefund()
}

func (this *TaraxaStateDB) CommitTransientState() *TransientState {
	ret := this.currentTransientState
	this.currentTransientState = newTransientState()
	this.lastCommittedSnapshot = this.transientStateLog.CurrentSnapshot()
	return ret
}

func (this *TaraxaStateDB) RevertToSnapshot(snapshotId int) {
	util.Assert(snapshotId > this.lastCommittedSnapshot)
	this.transientStateLog.RevertToSnapshot(snapshotId)
	this.stateDB.RevertToSnapshot(snapshotId)
}

func (this *TaraxaStateDB) Snapshot() int {
	snapshotId := this.stateDB.Snapshot()
	this.transientStateLog.Snapshot(snapshotId)
	return snapshotId
}

func (this *TaraxaStateDB) AddPreimage(hash common.Hash, val []byte) {
	this.stateDB.AddPreimage(hash, val)
}

func (this *TaraxaStateDB) Prepare(thash, bhash common.Hash, ti int) {
	this.stateDB.Prepare(thash, bhash, ti)
}

func (this *TaraxaStateDB) GetLogs(hash common.Hash) []*types.Log {
	return this.stateDB.GetLogs(hash)
}

func (this *TaraxaStateDB) Error() error {
	return this.stateDB.Error()
}

func (this *TaraxaStateDB) onGetAccount(addr common.Address) {
	this.conflictLogger(conflict_detector.GET, accountKey(addr))
}

func (this *TaraxaStateDB) onCreateOrDeleteAccount(addr common.Address) {
	this.conflictLogger(conflict_detector.SET, accountKey(addr))
}

func (this *TaraxaStateDB) onGetOrCreateAccount(addr common.Address) {
	this.conflictLogger(conflict_detector.DEFAULT_INITIALIZE, accountKey(addr))
}

func (this *TaraxaStateDB) onAccountRead(addr common.Address, key string) {
	this.onGetAccount(addr)
	if this.stateDB.Exist(addr) {
		this.conflictLogger(conflict_detector.GET, accountCompositeKey(addr, key))
	}
}

func (this *TaraxaStateDB) onAccountWrite(address common.Address, key string) {
	this.onGetOrCreateAccount(address)
	this.conflictLogger(conflict_detector.SET, accountCompositeKey(address, key))
}

func (this *TaraxaStateDB) onAccountEmptyCheck(addr common.Address) {
	this.conflictLogger(conflict_detector.GET, accountCompositeKey(addr, balance))
	this.conflictLogger(conflict_detector.GET, accountCompositeKey(addr, nonce))
	this.conflictLogger(conflict_detector.GET, accountCompositeKey(addr, code))
}

func accountKey(address common.Address) string {
	return address.Hex()
}

func accountCompositeKey(address common.Address, subKey string) string {
	return accountKey(address) + subKey
}

func newTransientState() *TransientState {
	ret := new(TransientState)
	ret.BalanceDeltas = make(map[common.Address]*big.Int)
	ret.NonceDeltas = make(map[common.Address]uint64)
	return ret
}
