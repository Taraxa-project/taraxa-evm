package state_dry_runner

import (
	"encoding/json"
	"fmt"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
)

type TraceRunner struct {
	db             state_db.DB
	get_block_hash vm.GetHashFunc
	dpos_api       *dpos.API
	get_reader     func(blk_n types.BlockNum) contract_storage.StorageReader
	chain_config   *chain_config.ChainConfig
}

func (self *TraceRunner) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	get_reader func(blk_n types.BlockNum) contract_storage.StorageReader,
	chain_config *chain_config.ChainConfig,
) *TraceRunner {
	self.db = db
	self.get_block_hash = get_block_hash
	self.dpos_api = dpos_api
	self.get_reader = get_reader
	self.chain_config = chain_config
	return self
}

func (self *TraceRunner) UpdateConfig(cfg *chain_config.ChainConfig) {
	self.chain_config = cfg
}

func (self *TraceRunner) Trace(blk *vm.Block, state_trxs *[]vm.Transaction, trxs *[]vm.Transaction, conf *vm.TracingConfig) []byte {
	if trxs == nil || blk == nil {
		return nil
	}

	blk_n := blk.Number
	// get state of previous block, so transactions from this block won't be applied yet
	if blk_n > 0 {
		blk_n -= 1
	}
	block_state := state_evm.GetBlockState(self.db, blk_n, len(*trxs))

	var evm vm.EVM
	evm.Init(self.get_block_hash, block_state, vm.DefaultOpts(), self.chain_config.EVMChainConfig, vm.Config{})
	evm.SetBlock(blk, self.chain_config.Hardforks.Rules(blk.Number))
	if self.dpos_api != nil {
		self.dpos_api.InitAndRegisterAllContracts(contract_storage.EVMStateStorage{block_state}, blk.Number, func(uint64) contract_storage.StorageReader { return block_state }, &evm, evm.RegisterPrecompiledContract)
	}

	for _, trx := range *state_trxs {
		evm.Main(&trx)
	}

	output := make([]any, len(*trxs))
	for index, trx := range *trxs {
		var tracer vm.Tracer
		if conf != nil {
			tracer = vm.NewOeTracer(conf)
		} else {
			// tracer = vm.NewStructLogger(config.LogConfig)
			tracer = vm.NewStructLogger(nil)
		}
		evm.UpdateVmConfig(vm.Config{Debug: true, Tracer: tracer})

		ret, _ := evm.Main(&trx)

		// Depending on the tracer type, format and return the output
		switch tracer := tracer.(type) {
		case *vm.StructLogger:
			failed := len(ret.ExecutionErr) != 0 || len(ret.ConsensusErr) != 0
			output[index] = ExecutionResult{
				Gas:         ret.GasUsed,
				Failed:      failed,
				ReturnValue: fmt.Sprintf("%x", ret.CodeRetval),
				StructLogs:  vm.FormatLogs(tracer.StructLogs()),
			}
		case *vm.OeTracer:
			tracer.SetRetCode(ret.CodeRetval)
			output[index] = tracer.GetResult()
		default:
			panic(fmt.Sprintf("bad tracer type %T", tracer))
		}
	}
	out, _ := json.Marshal(output)
	return out
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64            `json:"gas"`
	Failed      bool              `json:"failed"`
	ReturnValue string            `json:"returnValue"`
	StructLogs  []vm.StructLogRes `json:"structLogs"`
}
