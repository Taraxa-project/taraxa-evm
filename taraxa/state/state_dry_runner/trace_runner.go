package state_dry_runner

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

type TraceRunner struct {
	db             state_db.DB
	get_block_hash vm.GetHashFunc
	dpos_api       *dpos.API
	get_reader     func(types.BlockNum) dpos.Reader
	chain_config   *chain_config.ChainConfig
}

func (self *TraceRunner) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	get_reader func(types.BlockNum) dpos.Reader,
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

func (self *TraceRunner) Trace(blk *vm.Block, trx *vm.Transaction, conf *vm.TracingConfig) []byte {
	var evm_state state_evm.EVMState
	evm_state.Init(state_evm.Opts{
		NumTransactionsToBuffer: 1,
	})
	evm_state.SetInput(state_db.GetBlockState(self.db, blk.Number))
	// we don't need to specify nonce for eth_call. So set correct one
	trx.Nonce = bigutil.Add(evm_state.GetAccount(&trx.From).GetNonce(), big.NewInt(1))
	var evm vm.EVM
	var tracer vm.Tracer
	if conf != nil {
		tracer = vm.NewOeTracer(conf)
	} else {
		tracer = vm.NewStructLogger(nil)
	}

	// tracer = vm.NewStructLogger(config.LogConfig)
	evm.Init(self.get_block_hash, &evm_state, vm.Opts{}, self.chain_config.EVMChainConfig, vm.Config{Debug: true, Tracer: tracer})
	evm.SetBlock(blk /*, self.chain_config.EVMChainConfig.Rules(blk.Number)*/)
	if self.dpos_api != nil {
		self.dpos_api.NewContract(dpos.EVMStateStorage{&evm_state}, self.get_reader(blk.Number), &evm).Register(evm.RegisterPrecompiledContract)
	}

	ret := evm.Main(trx)

	// Depending on the tracer type, format and return the output
	switch tracer := tracer.(type) {
	case *vm.StructLogger:
		failed := len(ret.ExecutionErr) != 0 || len(ret.ConsensusErr) != 0
		out, _ := json.Marshal(ExecutionResult{
			Gas:         ret.GasUsed,
			Failed:      failed,
			ReturnValue: fmt.Sprintf("%x", ret.CodeRetval),
			StructLogs:  vm.FormatLogs(tracer.StructLogs()),
		})
		return out

	case *vm.OeTracer:
		tracer.SetRetCode(ret.CodeRetval)
		out, _ := json.Marshal(tracer.GetResult())
		return out
	default:
		panic(fmt.Sprintf("bad tracer type %T", tracer))
	}
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
