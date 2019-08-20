package vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/conflict_detector"
	"math/big"
)

const balance = "balance"
const code = "code"
const nonce = "nonce"

type OperationLogger func(conflict_detector.OperationType, conflict_detector.Key)

type OperationLoggingStateDB struct {
	StateDB
	OperationLogger conflict_detector.OperationLogger
}

func (this *OperationLoggingStateDB) CreateAccount(addr common.Address) {
	this.onCreateOrDeleteAccount(addr)
	if this.StateDB.Exist(addr) {
		this.OperationLogger(conflict_detector.SET, accountCompositeKey(addr, balance))
	}
	this.StateDB.CreateAccount(addr)
}

func (this *OperationLoggingStateDB) SubBalance(addr common.Address, value *big.Int) {
	this.onAddBalance(addr, value)
	this.StateDB.SubBalance(addr, value)
}

func (this *OperationLoggingStateDB) AddBalance(addr common.Address, value *big.Int) {
	this.onAddBalance(addr, value)
	if value.Sign() == 0 {
		this.onAccountEmptyCheck(addr)
	}
	this.StateDB.AddBalance(addr, value)
}

func (this *OperationLoggingStateDB) onAddBalance(addr common.Address, value *big.Int) {
	this.onGetOrCreateAccount(addr)
	if value.Sign() != 0 {
		this.OperationLogger(conflict_detector.ADD, accountCompositeKey(addr, balance))
	}
}

//func (this *OperationLoggingStateDB) SetBalance(addr common.Address, value *big.Int) {
//	this.onAccountWrite(addr, balance)
//	this.StateDB.SetBalance(addr, value)
//}

func (this *OperationLoggingStateDB) GetBalance(addr common.Address) *big.Int {
	this.onAccountRead(addr, balance)
	return this.StateDB.GetBalance(addr)
}

func (this *OperationLoggingStateDB) GetNonce(addr common.Address) uint64 {
	this.onAccountRead(addr, nonce)
	return this.StateDB.GetNonce(addr)
}

func (this *OperationLoggingStateDB) AddNonce(addr common.Address, val uint64) {
	this.onGetOrCreateAccount(addr)
	this.OperationLogger(conflict_detector.ADD, accountCompositeKey(addr, nonce))
	this.StateDB.AddNonce(addr, val)
}

func (this *OperationLoggingStateDB) SetCode(addr common.Address, val []byte) {
	this.onAccountWrite(addr, code)
	this.StateDB.SetCode(addr, val)
}

func (this *OperationLoggingStateDB) GetCodeHash(addr common.Address) common.Hash {
	this.onAccountRead(addr, code)
	return this.StateDB.GetCodeHash(addr)
}

func (this *OperationLoggingStateDB) GetCode(addr common.Address) []byte {
	this.onAccountRead(addr, code)
	return this.StateDB.GetCode(addr)
}

func (this *OperationLoggingStateDB) GetCodeSize(addr common.Address) int {
	this.onAccountRead(addr, code)
	return this.StateDB.GetCodeSize(addr)
}

func (this *OperationLoggingStateDB) Suicide(addr common.Address) bool {
	this.onGetAccount(addr)
	hasSuicided := this.StateDB.Suicide(addr)
	if hasSuicided {
		this.onCreateOrDeleteAccount(addr)
	}
	return hasSuicided
}

func (this *OperationLoggingStateDB) HasSuicided(addr common.Address) bool {
	this.onGetAccount(addr)
	return this.StateDB.HasSuicided(addr)
}

func (this *OperationLoggingStateDB) Exist(addr common.Address) bool {
	this.onGetAccount(addr)
	return this.StateDB.Exist(addr)
}

func (this *OperationLoggingStateDB) Empty(addr common.Address) bool {
	this.onGetAccount(addr)
	if this.StateDB.Exist(addr) {
		this.onAccountEmptyCheck(addr)
	}
	return this.StateDB.Empty(addr)
}

func (this *OperationLoggingStateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	this.onAccountRead(addr, hash.Hex())
	return this.StateDB.GetCommittedState(addr, hash)
}

func (this *OperationLoggingStateDB) GetState(addr common.Address, hash common.Hash) common.Hash {
	this.onAccountRead(addr, hash.Hex())
	return this.StateDB.GetState(addr, hash)
}

func (this *OperationLoggingStateDB) SetState(addr common.Address, hash common.Hash, value common.Hash) {
	this.onAccountWrite(addr, hash.Hex())
	this.StateDB.SetState(addr, hash, value)
}

func (this *OperationLoggingStateDB) onGetAccount(addr common.Address) {
	this.OperationLogger(conflict_detector.GET, accountKey(addr))
}

func (this *OperationLoggingStateDB) onCreateOrDeleteAccount(addr common.Address) {
	this.OperationLogger(conflict_detector.SET, accountKey(addr))
}

func (this *OperationLoggingStateDB) onGetOrCreateAccount(addr common.Address) {
	this.OperationLogger(conflict_detector.DEFAULT_INITIALIZE, accountKey(addr))
}

func (this *OperationLoggingStateDB) onAccountRead(addr common.Address, key string) {
	this.onGetAccount(addr)
	if this.StateDB.Exist(addr) {
		this.OperationLogger(conflict_detector.GET, accountCompositeKey(addr, key))
	}
}

func (this *OperationLoggingStateDB) onAccountWrite(address common.Address, key string) {
	this.onGetOrCreateAccount(address)
	this.OperationLogger(conflict_detector.SET, accountCompositeKey(address, key))
}

func (this *OperationLoggingStateDB) onAccountEmptyCheck(addr common.Address) {
	this.OperationLogger(conflict_detector.GET, accountCompositeKey(addr, balance))
	this.OperationLogger(conflict_detector.GET, accountCompositeKey(addr, nonce))
	this.OperationLogger(conflict_detector.GET, accountCompositeKey(addr, code))
}

func accountKey(address common.Address) string {
	return address.Hex()
}

func accountCompositeKey(address common.Address, subKey string) string {
	return accountKey(address) + "_" + subKey
}
