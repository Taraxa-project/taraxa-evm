package state_db

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

// TODO "touch" https://github.com/ethereum/eips/issues/158
// TODO move to lower levels e.g. state.StateDB
type TaraxaStateDB struct {
	conflictLogger conflict_detector.Logger
	stateDB        *state.StateDB
	balanceDeltas  map[common.Address]*big.Int
	nonceDeltas    map[common.Address]uint64
}

func (this *TaraxaStateDB) Init(commonDB *state.StateDB, conflictLogger conflict_detector.Logger) *TaraxaStateDB {
	this.stateDB = commonDB
	this.conflictLogger = conflictLogger
	this.balanceDeltas = make(map[common.Address]*big.Int)
	this.nonceDeltas = make(map[common.Address]uint64)
	return this
}

func (this *TaraxaStateDB) CreateAccount(addr common.Address) {
	this.onAccountWrite(addr)
	this.stateDB.CreateAccount(addr)
}

func (this *TaraxaStateDB) SubBalance(addr common.Address, value *big.Int) {
	//this.onAccountWrite(addr, balance)
	this.AddBalance(addr, new(big.Int).Neg(value))
}

func (this *TaraxaStateDB) AddBalance(addr common.Address, value *big.Int) {
	prev := this.balanceDeltas[addr]
	if prev == nil {
		prev = common.Big0
	}
	this.balanceDeltas[addr] = util.Sum(prev, value)
	//this.stateDB.AddBalance(addr, value)
}

func (this *TaraxaStateDB) GetBalance(addr common.Address) *big.Int {
	//this.onAccountRead(addr, balance)
	delta := this.balanceDeltas[addr]
	if delta == nil {
		delta = common.Big0
	}
	baseBalance := this.stateDB.GetBalance(addr)
	return util.Sum(baseBalance, delta)
}

func (this *TaraxaStateDB) HasBalance(address common.Address, amount *big.Int) bool {
	return true
}

func (this *TaraxaStateDB) GetNonce(addr common.Address) uint64 {
	//this.onAccountRead(addr, nonce)
	delta := this.nonceDeltas[addr]
	baseNonce := this.stateDB.GetNonce(addr)
	return baseNonce + delta
}

func (this *TaraxaStateDB) SetNonce(addr common.Address, value uint64) {
	this.onAccountWrite(addr, nonce)
	this.stateDB.SetNonce(addr, value)
}

func (this *TaraxaStateDB) AddNonce(addr common.Address, val uint64) {
	prev := this.nonceDeltas[addr]
	this.nonceDeltas[addr] = prev + val
	//this.stateDB.AddNonce(addr, val)
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
	this.onAccountRead(addr)
	hasSuicided := this.stateDB.Suicide(addr)
	if hasSuicided {
		// read write
		this.onAccountWrite(addr, balance)
	}
	return hasSuicided
}

func (this *TaraxaStateDB) HasSuicided(addr common.Address) bool {
	this.onAccountRead(addr)
	return this.stateDB.HasSuicided(addr)
}

func (this *TaraxaStateDB) Exist(addr common.Address) bool {
	this.onAccountRead(addr)
	return this.stateDB.Exist(addr)
}

func (this *TaraxaStateDB) Empty(addr common.Address) bool {
	this.onAccountRead(addr, nonce, balance, code)
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

func (this *TaraxaStateDB) RevertToSnapshot(pos int) {
	this.stateDB.RevertToSnapshot(pos)
}

func (this *TaraxaStateDB) Snapshot() int {
	return this.stateDB.Snapshot()
}

func (this *TaraxaStateDB) AddPreimage(hash common.Hash, val []byte) {
	this.stateDB.AddPreimage(hash, val)
}

func (this *TaraxaStateDB) onAccountRead(address common.Address, keys ...string) {
	accountKey := address.Hex()
	this.conflictLogger(conflict_detector.GET, accountKey)
	if len(keys) > 0 && this.stateDB.Exist(address) {
		for _, key := range keys {
			this.conflictLogger(conflict_detector.GET, accountKey+key)
		}
	}
}

func (this *TaraxaStateDB) onAccountWrite(address common.Address, keys ...string) {
	accountKey := address.Hex()
	this.onAccountRead(address)
	if len(keys) == 0 || !this.stateDB.Exist(address) {
		this.conflictLogger(conflict_detector.SET, accountKey)
	}
	for _, key := range keys {
		this.conflictLogger(conflict_detector.SET, accountKey+key)
	}
}
