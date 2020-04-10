package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

type ConcurrentScheduleGeneration struct {
	base_blk EVMStateInput
	evm_cfg  vm.EVMConfig
}

type TransactionWithHash struct {
	Hash common.Hash
	vm.Transaction
}

func (self *ConcurrentScheduleGeneration) SubmitTransactions(trxs ...TransactionWithHash) {

}

func (self *ConcurrentScheduleGeneration) Commit(trx_hashes ...common.Hash) (ret ConcurrentSchedule) {
	return
}
