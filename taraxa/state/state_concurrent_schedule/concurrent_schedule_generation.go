package state_concurrent_schedule

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

type ConcurrentScheduleGeneration struct {
	db             state_common.DB
	get_block_hash vm.GetHashFunc
	chain_cfg      state_common.EVMChainConfig
	curr_blk       vm.Block
}

func (self *ConcurrentScheduleGeneration) Init(
	db state_common.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.EVMChainConfig,
	curr_blk_num types.BlockNum,
) {
	self.db = db
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_cfg
	self.curr_blk.Number = curr_blk_num
}

func (self *ConcurrentScheduleGeneration) Begin(blk *vm.BlockWithoutNumber) {
	self.curr_blk.BlockWithoutNumber = *blk
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
	self.curr_blk.Number++
	return
}