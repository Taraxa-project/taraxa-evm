package state_dry_runner

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/accounts/abi"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/dpos/precompiled"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

type DryRunner struct {
	db             state_db.DB
	get_block_hash vm.GetHashFunc
	dpos_api       *dpos.API
	get_reader     func(blk_n types.BlockNum) contract_storage.StorageReader
	chain_config   *chain_config.ChainConfig
}

func (self *DryRunner) Init(
	db state_db.DB,
	get_block_hash vm.GetHashFunc,
	dpos_api *dpos.API,
	get_reader func(blk_n types.BlockNum) contract_storage.StorageReader,
	chain_config *chain_config.ChainConfig,
) *DryRunner {
	self.db = db
	self.get_block_hash = get_block_hash
	self.dpos_api = dpos_api
	self.get_reader = get_reader
	self.chain_config = chain_config
	return self
}

func (self *DryRunner) UpdateConfig(cfg *chain_config.ChainConfig) {
	self.chain_config = cfg
}

func (self *DryRunner) Apply(blk *vm.Block, trx *vm.Transaction) vm.ExecutionResult {
	var evm_state state_evm.EVMState
	evm_state.Init(state_evm.Opts{
		NumTransactionsToBuffer: 1,
	})
	evm_state.SetInput(state_db.GetBlockState(self.db, blk.Number))
	// we don't need to specify nonce for eth_call. So set correct one
	trx.Nonce = bigutil.Add(evm_state.GetAccount(&trx.From).GetNonce(), big.NewInt(1))
	var evm vm.EVM
	evm.Init(self.get_block_hash, &evm_state, vm.DefaultOpts(), self.chain_config.EVMChainConfig, vm.Config{})
	evm.SetBlock(blk, self.chain_config.Hardforks.Rules(blk.Number))
	if self.dpos_api != nil {
		self.dpos_api.InitAndRegisterAllContracts(contract_storage.EVMStateStorage{&evm_state}, blk.Number, self.get_reader, &evm, evm.RegisterPrecompiledContract)
	}

	ret, err := evm.Main(trx)
	if err == vm.ErrExecutionReverted {
		reason, unpack_err := abi.UnpackRevert(ret.CodeRetval)
		if unpack_err == nil {
			ret.ExecutionErr += util.ErrorString(": " + reason)
		}
	}
	return ret
}
