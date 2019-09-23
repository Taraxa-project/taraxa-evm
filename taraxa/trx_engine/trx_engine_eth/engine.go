package trx_engine_eth

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"sync"
)

type EthTrxEngine struct {
	*trx_engine_base.BaseTrxEngine
	EthTrxEngineConfig
	stateDB *state.StateDB
	mutex   sync.Mutex
}

func (this *EthTrxEngine) TransitionState(req *trx_engine.StateTransitionRequest) (ret *trx_engine.StateTransitionResult, err error) {
	defer util.Stringify(&err)
	defer concurrent.LockUnlock(&this.mutex)()
	ret = new(trx_engine.StateTransitionResult)
	if req.Block.Number.Sign() == 0 {
		this.ApplyGenesis()
		ret.StateRoot = this.GenesisBlock.Root()
		return
	}
	if this.stateDB == nil || this.calculateRoot(req.Block) != req.BaseStateRoot {
		if this.stateDB, err = state.New(req.BaseStateRoot, this.ReadDB); err != nil {
			return
		}
	}
	this.applyHardForks(req.Block)
	gasPool := new(core.GasPool).AddGas(uint64(req.Block.GasLimit))
	for i, tx := range req.Block.Transactions {
		if this.FreeGas {
			tx_cpy := *tx
			tx_cpy.GasPrice = new(hexutil.Big)
			tx_cpy.Gas = ^hexutil.Uint64(0) / 100000
			tx = &tx_cpy
		}
		this.stateDB.Prepare(tx.Hash, req.Block.Hash, i)
		txResult := this.BaseTrxEngine.ExecuteTransaction(&trx_engine_base.TransactionRequest{
			Transaction:      tx,
			BlockHeader:      &req.Block.BlockHeader,
			DB:               this.stateDB,
			OnEvmInstruction: vm.NoopExecutionController,
			GasPool:          gasPool,
			CheckNonce:       !this.DisableNonceCheck,
		})
		if err = txResult.ConsensusErr; err != nil {
			return
		}
		var intermediateRoot []byte
		if this.Genesis.Config.IsByzantium(req.Block.Number) {
			this.stateDB.Finalise(true)
		} else {
			intermediateRoot = this.calculateRoot(req.Block).Bytes()
		}
		ret.UsedGas += hexutil.Uint64(txResult.GasUsed)
		ethReceipt := types.NewReceipt(intermediateRoot, txResult.ContractErr != nil, uint64(ret.UsedGas))
		if tx.To == nil {
			ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, uint64(tx.Nonce))
		}
		ethReceipt.TxHash = tx.Hash;
		ethReceipt.GasUsed = txResult.GasUsed
		ethReceipt.Logs = this.stateDB.GetLogs(tx.Hash)
		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
		ret.Receipts = append(ret.Receipts, &trx_engine.TaraxaReceipt{
			ReturnValue:     txResult.EVMReturnValue,
			ContractError:   txResult.ContractErr,
			EthereumReceipt: ethReceipt,
		})
		ret.AllLogs = append(ret.AllLogs, ethReceipt.Logs...)
	}
	this.applyMinerReward(req.Block)
	ret.StateRoot, err = this.stateDB.Commit(this.Genesis.Config.IsEIP158(req.Block.Number))
	ret.UpdatedBalances = make(map[common.Address]*hexutil.Big)
	for k, v := range this.stateDB.GetAndResetUpdatedBalances() {
		ret.UpdatedBalances[k] = (*hexutil.Big)(v)
	}
	return
}

func (this *EthTrxEngine) applyMinerReward(block *trx_engine.Block) {
	if this.DisableMinerReward {
		return
	}
	var unclesMapped []*types.Header
	for _, uncle := range block.UncleBlocks {
		unclesMapped = append(unclesMapped, &types.Header{Number: uncle.Number.ToInt(), Coinbase: uncle.Miner})
	}
	ethash.AccumulateRewards(
		this.Genesis.Config,
		this.stateDB,
		&types.Header{Number: block.Number, Coinbase: block.Miner},
		unclesMapped)
}

func (this *EthTrxEngine) calculateRoot(block *trx_engine.Block) common.Hash {
	return this.stateDB.IntermediateRoot(this.Genesis.Config.IsEIP158(block.Number))
}

func (this *EthTrxEngine) applyHardForks(block *trx_engine.Block) (stateChanged bool) {
	chainConfig := this.Genesis.Config
	DAOForkBlock := chainConfig.DAOForkBlock
	if chainConfig.DAOForkSupport && DAOForkBlock != nil && DAOForkBlock.Cmp(block.Number) == 0 {
		misc.ApplyDAOHardFork(this.stateDB)
		return true
	}
	return false
}
