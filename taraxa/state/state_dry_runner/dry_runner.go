package state_dry_runner

import (
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_historical"
)

type DryRunner struct {
	db             state_historical.DB
	get_block_hash vm.GetHashFunc
	chain_cfg      state_common.EVMChainConfig
}

func (self *DryRunner) Init(
	db state_historical.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.EVMChainConfig,
) {
	self.db = db
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_cfg
}

func (self *DryRunner) Apply(blk *vm.Block, trx *vm.Transaction, opts *vm.ExecutionOptions) (ret vm.ExecutionResult) {
	if opts == nil {
		opts = &self.chain_cfg.ExecutionOptions
	}
	evm_cfg := vm.NewEVMConfig(self.get_block_hash, blk, self.chain_cfg.ETHChainConfig.Rules(blk.Number), *opts)
	var evm_state state_evm.EVMState
	evm_state.Init(self.db.AtBlock(blk.Number), state_evm.CacheOpts{
		AccountsPrealloc:      32,
		DirtyAccountsPrealloc: 16,
	})
	ret = vm.Main(&evm_cfg, &evm_state, trx)
	return
}
