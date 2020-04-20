package state_concurrent_schedule

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type ConcurrentScheduleGeneration struct {
	db             state_common.DB
	get_block_hash vm.GetHashFunc
	chain_cfg      state_common.EvmChainConfig
	curr_blk       vm.Block
}

func (self *ConcurrentScheduleGeneration) Init(
	db state_common.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.EvmChainConfig,
) {
	self.db = db
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_cfg
}

func (self *ConcurrentScheduleGeneration) Begin(blk vm.Block) {
	self.curr_blk = blk
}

type TransactionWithHash struct {
	Hash common.Hash
	vm.Transaction
}

func (self *ConcurrentScheduleGeneration) SubmitTransactions(trxs ...TransactionWithHash) {

}

type ConcurrentSchedule struct {
	ParallelTransactions []state_common.TxIndex
}

func (self *ConcurrentScheduleGeneration) Commit(trx_hashes ...common.Hash) (ret ConcurrentSchedule) {
	return
}
