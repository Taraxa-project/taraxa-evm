package state_dry_runner

import (
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
)

type DryRunner struct {
	db             state_db.ReadOnlyDB
	get_block_hash vm.GetHashFunc
	dpos_api       *dpos.API
	exec_cfg       state_common.ExecutionConfig
}

func (self *DryRunner) Init(
	db state_db.ReadOnlyDB,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	exec_cfg state_common.ExecutionConfig,
) *DryRunner {
	self.db = db
	self.get_block_hash = get_block_hash
	self.dpos_api = dpos_api
	self.exec_cfg = exec_cfg
	return self
}

func (self *DryRunner) Apply(blk *vm.Block, trx *vm.Transaction, opts *vm.ExecutionOpts) vm.ExecutionResult {
	if opts == nil {
		opts = &self.exec_cfg.Options
	}
	var evm_state state_evm.EVMState
	evm_state.Init(state_evm.Opts{
		NumTransactionsToBuffer: 1,
	})
	evm_state.SetInput(state_db.ExtendedReader{self.db.GetBlockState(blk.Number)})
	var evm vm.EVM
	evm.Init(self.get_block_hash, &evm_state, vm.Opts{})
	evm.SetBlock(blk.Number, &blk.BlockInfo, self.exec_cfg.ETHForks.Rules(blk.Number))
	if self.dpos_api != nil {
		self.dpos_api.NewContract(dpos.EVMStateStorage{&evm_state}).Register(evm.RegisterPrecompiledContract)
	}
	return evm.Main(trx, *opts)
}
