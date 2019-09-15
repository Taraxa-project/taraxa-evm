package trx_engine_taraxa

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
)

type TransactionState struct {
	origin               *OriginState
	loggingContext       *LoggingContext
	accounts             map[common.Address]*TransactionAccount
	dirtyAccounts        AccountSet
	dirtyAccountsByField [AccountField_count]AccountSet
	reverted             bool
	refund               uint64
	preimages            Preimages
	logs                 Logs
	onError              util.ErrorHandler
}
type LoggingContext = struct {
	TxIndex   trx_engine.TxIndex
	TxHash    common.Hash
	BlockHash common.Hash
}

func NewTransactionState(origin *OriginState, loggingCtx *LoggingContext, onErr util.ErrorHandler) *TransactionState {
	return &TransactionState{
		origin:         origin,
		loggingContext: loggingCtx,
		accounts:       make(map[common.Address]*TransactionAccount),
		dirtyAccounts:  make(AccountSet),
		onError:        onErr,
		preimages:      make(Preimages),
	}
}

func (this *TransactionState) GetBalance(addr common.Address) *big.Int {
	if acc := this.getAccount(addr); acc != nil {
		return acc.balance
	}
	return common.Big0
}

func (this *TransactionState) BalanceEQ(addr common.Address, value *big.Int) bool {
	return this.GetBalance(addr).Cmp(value) == 0
}

func (this *TransactionState) AssertBalanceGTE(addr common.Address, value *big.Int) bool {
	return this.GetBalance(addr).Cmp(value) >= 0
}

func (this *TransactionState) GetNonce(addr common.Address) uint64 {
	if acc := this.getAccount(addr); acc != nil {
		return acc.nonce
	}
	return 0
}

func (this *TransactionState) NonceEQ(addr common.Address, value uint64) bool {
	return this.GetNonce(addr) == value
}

func (this *TransactionState) GetCodeHash(addr common.Address) (ret common.Hash) {
	if acc := this.getAccount(addr); acc != nil {
		return acc.GetCodeHash()
	}
	return
}

func (this *TransactionState) GetCode(addr common.Address) (ret []byte) {
	if acc := this.getAccount(addr); acc != nil {
		val, err := acc.GetCode()
		if err == nil {
			return val
		}
		this.onError(err)
	}
	return
}

func (this *TransactionState) GetCodeSize(addr common.Address) (ret int) {
	if acc := this.getAccount(addr); acc != nil {
		val, err := acc.GetCodeSize()
		if err == nil {
			return val
		}
		this.onError(err)
	}
	return
}

func (this *TransactionState) GetCommittedState(addr common.Address, key common.Hash) (ret common.Hash) {
	if acc := this.getAccount(addr); acc != nil {
		val, err := acc.GetOriginStorage(key)
		if err == nil {
			return val
		}
		this.onError(err)
	}
	return
}

func (this *TransactionState) GetState(addr common.Address, key common.Hash) (ret common.Hash) {
	if acc := this.getAccount(addr); acc != nil {
		val, err := acc.GetStorage(key)
		if err == nil {
			return val
		}
		this.onError(err)
	}
	return
}

func (this *TransactionState) HasSuicided(addr common.Address) bool {
	if acc := this.getAccount(addr); acc != nil {
		return acc.hasSuicided
	}
	return false
}

func (this *TransactionState) Exist(addr common.Address) bool {
	return this.getAccount(addr) != nil
}

func (this *TransactionState) Empty(addr common.Address) bool {
	if acc := this.getAccount(addr); acc != nil {
		return acc.IsEmpty()
	}
	return true
}

func (this *TransactionState) CreateAccount(addr common.Address) {
	prev := this.getAccount(addr)
	new := this.createOrResetAccount(addr)
	if prev != nil {
		new.setBalance(prev.balance)
	}
}

func (this *TransactionState) SubBalance(addr common.Address, value *big.Int) {
	acc := this.getOrCreateAccount(addr)
	if value.Sign() == 0 {
		return
	}
	acc.setBalance(new(big.Int).Sub(acc.balance, value))
}

func (this *TransactionState) AddBalance(addr common.Address, value *big.Int) {
	acc := this.getOrCreateAccount(addr)
	if value.Sign() == 0 {
		if acc.IsEmpty() {
			acc.markDirty()
		}
		return
	}
	acc.setBalance(new(big.Int).Add(acc.balance, value))
}

func (this *TransactionState) Transfer(from, to common.Address, value *big.Int) {
	this.SubBalance(from, value)
	this.AddBalance(to, value)
}

func (this *TransactionState) IncrementNonce(addr common.Address) {
	this.getOrCreateAccount(addr).incNonce()
}

func (this *TransactionState) SetCode(addr common.Address, value []byte) {
	this.getOrCreateAccount(addr).setCode(value)
}

func (this *TransactionState) SetState(addr common.Address, key, value common.Hash) {
	this.getOrCreateAccount(addr).setStorage(key, value)
}

func (this *TransactionState) Suicide(addr, newAddr common.Address) {
	acc := this.getAccount(addr)
	if acc == nil {
		this.AddBalance(newAddr, common.Big0)
		return
	}
	this.AddBalance(newAddr, acc.balance)
	acc.hasSuicided = true
	acc.setBalance(common.Big0)
}

func (this *TransactionState) AddRefund(value uint64) {
	this.refund += value
}

func (this *TransactionState) SubRefund(value uint64) {
	this.refund -= value
}

func (this *TransactionState) GetRefund() uint64 {
	return this.refund
}

func (this *TransactionState) AddLog(log *types.Log) {
	log.TxHash = this.loggingContext.TxHash
	log.BlockHash = this.loggingContext.BlockHash
	log.TxIndex = uint(this.loggingContext.TxIndex)
	//log.Index = len(this.logs) TODO global index
	this.logs = append(this.logs, log)
}

func (this *TransactionState) AddPreimage(hash common.Hash, value []byte) {
	if _, ok := this.preimages[hash]; ok {
		return
	}
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	this.preimages[hash] = valueCopy
}

func (this *TransactionState) RevertToSnapshot(revision int) {
	this.reverted = true
}

func (this *TransactionState) Snapshot() int {
	return 0
}

func (this *TransactionState) getAccount(addr common.Address) (ret *TransactionAccount) {
	if acc, present := this.accounts[addr]; present {
		return acc
	}
	originAcc, err := this.origin.LoadAccount(addr);
	if err != nil {
		this.onError(err)
		return
	}
	if originAcc == nil {
		this.accounts[addr] = nil
		return
	}
	ret = this.newAccount(addr)
	ret.origin = originAcc
	ret.balance = originAcc.Balance
	ret.nonce = originAcc.Nonce
	if originAcc.HasCode() {
		ret.code = &Code{Hash: originAcc.GetCodeHash()}
	}
	this.accounts[addr] = ret
	return
}

func (this *TransactionState) getOrCreateAccount(addr common.Address) *TransactionAccount {
	if acc := this.getAccount(addr); acc != nil {
		return acc
	}
	return this.createOrResetAccount(addr)
}

func (this *TransactionState) createOrResetAccount(addr common.Address) *TransactionAccount {
	acc := this.newAccount(addr)
	acc.markDirty()
	this.accounts[addr] = acc
	return acc
}

func (this *TransactionState) newAccount(addr common.Address) *TransactionAccount {
	return &TransactionAccount{
		addr:         addr,
		state:        this,
		storage:      make(Storage),
		dirtyStorage: make(StorageKeySet),
		balance:      common.Big0,
	}
}

type TransactionAccount struct {
	origin       *OriginAccount
	state        *TransactionState
	addr         common.Address
	balance      *big.Int
	nonce        uint64
	code         *Code
	storage      Storage
	hasSuicided  bool
	dirty        bool
	dirtyFields  AccountFieldSet
	dirtyStorage StorageKeySet
}

func (this *TransactionAccount) IsEmpty() bool {
	return this.nonce == 0 && this.balance.Sign() == 0 && !this.HasCode()
}

func (this *TransactionAccount) GetStorage(key common.Hash) (ret common.Hash, err error) {
	if cell := this.storage[key]; cell != nil {
		if cell.Value != nil {
			return *cell.Value, nil
		}
		if cell.OriginValue != nil {
			cell.Value = cell.OriginValue
			return *cell.OriginValue, nil
		}
	}
	return this.GetOriginStorage(key)
}

func (this *TransactionAccount) GetOriginStorage(key common.Hash) (ret common.Hash, err error) {
	if this.origin == nil || !this.origin.HasStorage() {
		return
	}
	cell := this.getOrCreateStorageCell(key)
	if cell.OriginValue == nil {
		if cell.OriginValue, err = this.origin.GetStorage(key); err != nil {
			return
		}
	}
	return *cell.OriginValue, nil
}

func (this *TransactionAccount) HasCode() bool {
	return this.code != nil
}

func (this *TransactionAccount) GetCodeHash() common.Hash {
	if !this.HasCode() {
		return EmptyCodeHash
	}
	return this.code.Hash
}

func (this *TransactionAccount) GetCode() (ret []byte, err error) {
	if !this.HasCode() {
		return
	}
	if this.code.Value == nil {
		if this.code.Value, err = this.origin.GetCode(); err != nil {
			return
		}
		this.code.Size = len(this.code.Value)
	}
	return this.code.Value, nil
}

func (this *TransactionAccount) GetCodeSize() (ret int, err error) {
	if !this.HasCode() {
		return
	}
	if this.code.Size == 0 {
		if this.code.Size, err = this.origin.GetCodeSize(); err != nil {
			return
		}
	}
	return this.code.Size, nil
}

func (this *TransactionAccount) getOrCreateStorageCell(key common.Hash) *StorageCell {
	if cell, present := this.storage[key]; present {
		return cell
	}
	cell := new(StorageCell)
	this.storage[key] = cell
	return cell
}

func (this *TransactionAccount) markDirty() {
	if !this.dirty {
		this.dirty = true
		this.state.dirtyAccounts[this.addr] = true
	}
}

func (this *TransactionAccount) markFieldDirty(field AccountField) {
	this.markDirty()
	if !this.dirtyFields[field] {
		this.dirtyFields[field] = true
		this.state.dirtyAccountsByField[field][this.addr] = true
	}
}

func (this *TransactionAccount) setBalance(val *big.Int) {
	this.markFieldDirty(balance)
	this.balance = val
}

func (this *TransactionAccount) incNonce() {
	this.markFieldDirty(nonce)
	this.nonce++
}

func (this *TransactionAccount) setCode(val []byte) {
	this.markFieldDirty(code)
	if len(val) == 0 {
		this.code = nil
		return
	}
	this.code = &Code{
		Hash:  crypto.Keccak256Hash(val),
		Value: val,
		Size:  len(val),
	}
}

func (this *TransactionAccount) setStorage(key common.Hash, value common.Hash) {
	this.markFieldDirty(storage)
	this.dirtyStorage[key] = true
	this.getOrCreateStorageCell(key).Value = &value
}
