package conflict_tracking

import "github.com/Taraxa-project/taraxa-evm/common"

type Conflicts struct {
	accounts                map[common.Address]*AccountCell
	conflictingTransactions map[TxId]DUMMY
}

func (this *Conflicts) Init() *Conflicts {
	this.accounts = make(map[common.Address]*AccountCell)
	this.conflictingTransactions = make(map[TxId]DUMMY)
	return this
}

func (this *Conflicts) getAccount(addr common.Address) *AccountCell {
	if account, present := this.accounts[addr]; present {
		return account;
	}
	accountCell := new(AccountCell).Constructor()
	this.accounts[addr] = accountCell
	return accountCell
}

func (this *Conflicts) GetConflictingTransactions() []TxId {
	result := make([]TxId, 0, len(this.conflictingTransactions))
	for txId, _ := range this.conflictingTransactions {
		result = append(result, txId)
	}
	return result
}
