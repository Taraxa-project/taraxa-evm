package state_dry_runner

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
	dpos_2 "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos_2.0/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
)

type DryRunner struct {
	db               state_db.DB
	get_block_hash   vm.GetHashFunc
	dpos_api         *dpos.API
	dpos2_api        *dpos_2.API
	get_reader 		 func(types.BlockNum) dpos_2.Reader
	chain_config     *chain_config.ChainConfig
}

func (self *DryRunner) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	dpos2_api *dpos_2.API,
	get_reader func(types.BlockNum) dpos_2.Reader,
	chain_config *chain_config.ChainConfig,
) *DryRunner {
	self.db = db
	self.get_block_hash = get_block_hash
	self.dpos_api = dpos_api
	self.dpos2_api = dpos2_api
	self.get_reader = get_reader
	self.chain_config = chain_config
	return self
}

func (self *DryRunner) UpdateConfig(cfg *chain_config.ChainConfig) {
	self.chain_config = cfg
}

func (self *DryRunner) Apply(blk *vm.Block, trx *vm.Transaction, opts *vm.ExecutionOpts) vm.ExecutionResult {
	if opts == nil {
		opts = &self.chain_config.ExecutionOptions
	}
	var evm_state state_evm.EVMState
	evm_state.Init(state_evm.Opts{
		NumTransactionsToBuffer: 1,
	})
	evm_state.SetInput(state_db.GetBlockState(self.db, blk.Number))
	var evm vm.EVM
	evm.Init(self.get_block_hash, &evm_state, vm.Opts{})
	evm.SetBlock(blk, self.chain_config.ETHChainConfig.Rules(blk.Number))
	if self.dpos_api != nil {
		self.dpos_api.NewContract(dpos.EVMStateStorage{&evm_state}).Register(evm.RegisterPrecompiledContract)
	}
	if self.dpos2_api != nil {
		self.dpos2_api.NewContract(dpos.EVMStateStorage{&evm_state}, self.get_reader(blk.Number)).Register(evm.RegisterPrecompiledContract)
	}
	return evm.Main(trx, *opts)
}
