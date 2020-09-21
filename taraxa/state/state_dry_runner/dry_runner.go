package state_dry_runner

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_historical"
)

type DryRunner struct {
	db             state_historical.DB
	get_block_hash vm.GetHashFunc
	exec_cfg       state_config.ExecutionConfig
}

func (self *DryRunner) Init(db state_historical.DB, get_block_hash vm.GetHashFunc, exec_cfg state_config.ExecutionConfig) {
	self.db = db
	self.get_block_hash = get_block_hash
	self.exec_cfg = exec_cfg
}

func (self *DryRunner) Apply(blk_num types.BlockNum, blk *vm.BlockWithoutNumber, trx *vm.Transaction, opts *vm.ExecutionOptions) (ret vm.ExecutionResult) {
	if opts == nil {
		opts = &self.exec_cfg.Options
	}
	blk_r, db_tx := self.db.ReadBlock(blk_num)
	defer db_tx.NotifyDoneReading()
	var evm_state state_evm.EVMState
	evm_state.Init(blk_r, state_evm.CacheOpts{
		AccountBufferSize: 32,
		RevertLogSize:     16,
	})
	var evm vm.EVM
	evm.SetGetHash(self.get_block_hash)
	evm.SetState(&evm_state)
	evm.SetBlock(blk_num, blk)
	evm.SetRules(self.exec_cfg.ETHForks.Rules(blk_num))
	return evm.Main(trx, *opts)
}
