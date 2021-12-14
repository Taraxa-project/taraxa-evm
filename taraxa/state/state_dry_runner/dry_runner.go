package state_dry_runner

import (
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
)

type DryRunner struct {
	db             state_db.DB
	get_block_hash vm.GetHashFunc
	dpos_api       *dpos.API
	chain_cfg      ChainConfig
}
type ChainConfig struct {
	ETHChainConfig   params.ChainConfig
	ExecutionOptions vm.ExecutionOpts
}

func (self *DryRunner) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	chain_cfg ChainConfig,
) *DryRunner {
	self.db = db
	self.get_block_hash = get_block_hash
	self.dpos_api = dpos_api
	self.chain_cfg = chain_cfg
	return self
}

func (self *DryRunner) Apply(blk *vm.Block, trx *vm.Transaction, opts *vm.ExecutionOpts) vm.ExecutionResult {
	if opts == nil {
		opts = &self.chain_cfg.ExecutionOptions
	}
	var evm_state state_evm.EVMState
	evm_state.Init(state_evm.Opts{
		NumTransactionsToBuffer: 1,
	})
	evm_state.SetInput(state_db.GetBlockState(self.db, blk.Number))
	var evm vm.EVM
	evm.Init(self.get_block_hash, &evm_state, vm.Opts{})
	evm.SetBlock(blk, self.chain_cfg.ETHChainConfig.Rules(blk.Number))
	if self.dpos_api != nil {
		self.dpos_api.NewContract(dpos.EVMStateStorage{&evm_state}).Register(evm.RegisterPrecompiledContract)
	}
	return evm.Main(trx, *opts, nil)
}
