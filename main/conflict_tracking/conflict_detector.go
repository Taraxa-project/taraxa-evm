package conflict_tracking

import "github.com/Taraxa-project/taraxa-evm/common"

type ConflictDetector struct {
	accounts                map[common.Address]*AccountCell
	conflictingTransactions map[TxId]DUMMY
}

func (this *ConflictDetector) Init() *ConflictDetector {
	this.accounts = make(map[common.Address]*AccountCell)
	this.conflictingTransactions = make(map[TxId]DUMMY)
	return this
}

func (this *ConflictDetector) getAccount(addr common.Address) *AccountCell {
	if account, present := this.accounts[addr]; present {
		return account;
	}
	accountCell := new(AccountCell).Constructor()
	this.accounts[addr] = accountCell
	return accountCell
}

func (this *ConflictDetector) Reset() []TxId {
	result := make([]TxId, 0, len(this.conflictingTransactions))
	for txId, _ := range this.conflictingTransactions {
		result = append(result, txId)
	}
	return result
}

func (this *ConflictDetector) InConflict(id TxId) bool {

}
