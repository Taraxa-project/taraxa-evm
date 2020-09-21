package state_evm

import (
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type Dirties struct {
	accounts       []*Account
	commit_version uint64
}

func (self *Dirties) Init(cap uint) *Dirties {
	self.accounts = make([]*Account, 0, cap)
	return self
}

func (self *Dirties) append(acc *Account) {
	self.accounts = append(self.accounts, acc)
}

func (self *Dirties) set_commit_version(v uint64) {
	self.commit_version = v
}

func (self *Dirties) ForEachMutationWithDuplicates(cb func(AccountMutation)) {
	for _, acc := range self.accounts {
		if !acc.deleted {
			cb(acc.sink)
		}
	}
}

func (self *Dirties) cleanup() {
	for _, acc := range self.accounts {
		if acc.upd_version <= self.commit_version {
			acc.unload()
		}
	}
	bin.ZFill_3(unsafe.Pointer(&self.accounts), unsafe.Sizeof(self.accounts[0]))
	self.commit_version, self.accounts = 0, self.accounts[:0]
}
