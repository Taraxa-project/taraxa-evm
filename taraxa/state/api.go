package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type API struct {
	db                         DB
	get_block_hash             vm.GetHashFunc
	chain_cfg                  params.ChainConfig
	execution_opts             vm.ExecutionOptions
	state_transition_service   StateTransitionService
	concur_schedule_generation ConcurrentScheduleGeneration
	util.InitFlag
}

func (self *API) I(
	db DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg params.ChainConfig,
	execution_opts vm.ExecutionOptions,
	disable_block_rewards bool,
	last_blk_num types.BlockNum,
	last_root_hash *common.Hash,
	main_trie_writer_opts trie.TrieWriterOpts,
) *API {
	self.InitOnce()
	self.db = db
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_cfg
	self.execution_opts = execution_opts
	self.state_transition_service.I(
		db,
		last_blk_num,
		get_block_hash,
		chain_cfg,
		execution_opts,
		disable_block_rewards,
		last_root_hash,
		main_trie_writer_opts,
	)
	return self
}

func (self *API) TransitionState() {

}

func (self *API) BeginConcurrentSchedule(target_blk *vm.Block) {
	self.concur_schedule_generation = ConcurrentScheduleGeneration{
		&self.state_transition_service.last_blk,
		vm.NewEVMConfig(self.get_block_hash, target_blk, self.chain_cfg.Rules(target_blk.Number), self.execution_opts),
	}
}

func (self *API) DryRun(blk *vm.Block, trx *vm.Transaction, opts vm.ExecutionOptions) (ret vm.ExecutionResult) {
	evm_cfg := vm.NewEVMConfig(self.get_block_hash, blk, self.chain_cfg.Rules(blk.Number), opts)
	evm_state := NewEVMState(&BlockState{self.db, blk.Number}, EvmStateOpts{
		AccountCacheSize:      32,
		DirtyAccountCacheSize: 16,
	})
	ret = vm.Main(&evm_cfg, &evm_state, trx)
	return
}

func (self *API) GetStateTransitionService() *StateTransitionService {
	return &self.state_transition_service
}

func (self *API) GetBlockState(blk_num types.BlockNum) (ret BlockState) {
	ret = BlockState{self.db, blk_num}
	return
}
