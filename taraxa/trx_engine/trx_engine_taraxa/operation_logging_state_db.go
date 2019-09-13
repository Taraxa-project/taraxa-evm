package trx_engine_taraxa

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/conflict_detector"
	"math/big"
)

type StateDBForConflictDetection struct {
	vm.StateDB
	EIP158       bool
	LogOperation conflict_detector.OperationLogger
}

func (this *StateDBForConflictDetection) Transfer(from common.Address, to common.Address, value *big.Int) {
	this.SubBalance(from, value)
	this.AddBalance(to, value)
}

func (this *StateDBForConflictDetection) BalanceEQ(addr common.Address, value *big.Int) bool {
	return this.GetBalance(addr).Cmp(value) == 0
}

func (this *StateDBForConflictDetection) AssertBalanceGTE(addr common.Address, value *big.Int) bool {
	return true
}

func (this *StateDBForConflictDetection) NonceEQ(addr common.Address, value uint64) bool {
	return this.GetNonce(addr) == value
}

func (this *StateDBForConflictDetection) CreateAccount(addr common.Address) {
	this.log(conflict_detector.GET, &addr)
	this.log(conflict_detector.SET, &addr)
	if this.StateDB.Exist(addr) {
		this.log(conflict_detector.GET, &addr, balance)
		this.log(conflict_detector.SET, &addr, balance)
	} else {
		this.log(conflict_detector.SET, &addr, nonce)
	}
	this.StateDB.CreateAccount(addr)
}

func (this *StateDBForConflictDetection) SubBalance(addr common.Address, value *big.Int) {
	this.logAddOrSubBalance(&addr, false, value.Sign() == 0)
	this.StateDB.SubBalance(addr, value)
}

func (this *StateDBForConflictDetection) AddBalance(addr common.Address, value *big.Int) {
	this.logAddOrSubBalance(&addr, true, value.Sign() == 0)
	this.StateDB.AddBalance(addr, value)
}

func (this *StateDBForConflictDetection) GetBalance(addr common.Address) *big.Int {
	this.logGetAccountField(&addr, balance)
	return this.StateDB.GetBalance(addr)
}

func (this *StateDBForConflictDetection) GetNonce(addr common.Address) uint64 {
	this.logGetAccountField(&addr, nonce)
	return this.StateDB.GetNonce(addr)
}

func (this *StateDBForConflictDetection) IncrementNonce(addr common.Address) {
	this.logGetOrCreateAccount(&addr)
	this.log(conflict_detector.ADD, &addr, nonce)
	this.StateDB.IncrementNonce(addr)
}

func (this *StateDBForConflictDetection) SetCode(addr common.Address, val []byte) {
	this.logSetAccountField(&addr, code)
	this.StateDB.SetCode(addr, val)
}

func (this *StateDBForConflictDetection) GetCodeHash(addr common.Address) common.Hash {
	this.logGetAccountField(&addr, code)
	return this.StateDB.GetCodeHash(addr)
}

func (this *StateDBForConflictDetection) GetCode(addr common.Address) []byte {
	this.logGetAccountField(&addr, code)
	return this.StateDB.GetCode(addr)
}

func (this *StateDBForConflictDetection) GetCodeSize(addr common.Address) int {
	this.logGetAccountField(&addr, code)
	return this.StateDB.GetCodeSize(addr)
}

func (this *StateDBForConflictDetection) Suicide(addr common.Address, newAddr common.Address) {
	if this.logGetAccountField(&addr, balance) {
		this.log(conflict_detector.SET, &addr, balance)
		this.logAddOrSubBalance(&newAddr, true, this.StateDB.GetBalance(addr).Sign() == 0)
		this.log(conflict_detector.DELETE, &addr)
	} else {
		this.logAddOrSubBalance(&newAddr, true, true)
	}
	this.StateDB.Suicide(addr, newAddr)
}

func (this *StateDBForConflictDetection) HasSuicided(addr common.Address) bool {
	this.logGetAccount(&addr)
	return this.StateDB.HasSuicided(addr)
}

func (this *StateDBForConflictDetection) Exist(addr common.Address) bool {
	this.logGetAccount(&addr)
	return this.StateDB.Exist(addr)
}

func (this *StateDBForConflictDetection) Empty(addr common.Address) bool {
	this.logGetAccount(&addr)
	if !this.StateDB.Exist(addr) {
		return true
	}
	this.logAndCheckIfEmpty(&addr)
	return this.StateDB.Empty(addr)
}

func (this *StateDBForConflictDetection) GetCommittedState(addr common.Address, location common.Hash) common.Hash {
	this.logGetAccountField(&addr, location)
	return this.StateDB.GetCommittedState(addr, location)
}

func (this *StateDBForConflictDetection) GetState(addr common.Address, location common.Hash) common.Hash {
	this.logGetAccountField(&addr, location)
	return this.StateDB.GetState(addr, location)
}

func (this *StateDBForConflictDetection) SetState(addr common.Address, location common.Hash, value common.Hash) {
	this.logSetAccountField(&addr, location)
	this.StateDB.SetState(addr, location, value)
}

func (this *StateDBForConflictDetection) logAddOrSubBalance(addr *common.Address, isAdd, isZero bool) {
	this.logGetOrCreateAccount(addr)
	if !isZero {
		this.log(conflict_detector.ADD, addr, balance)
	} else if isAdd && this.logAndCheckIfEmpty(addr) && this.EIP158 {
		this.log(conflict_detector.DELETE, addr)
	}
}

func (this *StateDBForConflictDetection) logGetAccount(addr *common.Address) {
	this.log(conflict_detector.GET, addr)
}

func (this *StateDBForConflictDetection) logGetAccountField(addr *common.Address, field interface{}) (accExists bool) {
	this.logGetAccount(addr)
	if this.StateDB.Exist(*addr) {
		this.log(conflict_detector.GET, addr, field)
		return true
	}
	return false
}

func (this *StateDBForConflictDetection) logGetOrCreateAccount(addr *common.Address) {
	if this.Exist(*addr) {
		this.log(conflict_detector.GET, addr)
	} else {
		this.log(conflict_detector.SET, addr)
	}
}

func (this *StateDBForConflictDetection) logSetAccountField(addr *common.Address, field interface{}) {
	this.logGetOrCreateAccount(addr)
	this.log(conflict_detector.SET, addr, field)
}

func (this *StateDBForConflictDetection) logAndCheckIfEmpty(addr *common.Address) bool {
	this.log(conflict_detector.GET, addr, balance, nonce, code)
	return this.StateDB.Empty(*addr)
}

type AccountKey = common.Address

type AccountFieldKey struct {
	Address common.Address
	field   AccountField
}

type AccountStorageKey struct {
	Address  common.Address
	Location common.Hash
}

func (this *StateDBForConflictDetection) log(
	opType conflict_detector.OperationType,
	addr *common.Address,
	fields ...interface{}) {
	if len(fields) == 0 {
		this.LogOperation(opType, *addr)
		return
	}
	for _, field := range fields {
		switch field := field.(type) {
		case AccountField:
			this.LogOperation(opType, AccountFieldKey{*addr, field})
		case *common.Hash:
			this.LogOperation(opType, AccountStorageKey{*addr, *field})
		default:
			panic("unknown type")
		}
	}
}
