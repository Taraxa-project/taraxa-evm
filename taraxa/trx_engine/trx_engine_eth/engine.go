package trx_engine_eth

import (
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
)

type EthTrxEngine struct {
	*trx_engine_base.BaseTrxEngine
	EthTrxEngineConfig
}

func (this *EthTrxEngine) TransitionState(req *trx_engine.StateTransitionRequest) (ret *trx_engine.StateTransitionResult, err error) {
	defer util.Stringify(&err)
	ret = new(trx_engine.StateTransitionResult)
	block := req.Block
	if block.Number.Sign() == 0 {
		this.ApplyGenesis()
		ret.StateRoot = this.GenesisBlock.Root()
		return
	}
	var stateDB *state.StateDB
	if stateDB, err = state.New(req.BaseStateRoot, this.ReadDB); err != nil {
		return
	}
	chainConfig := this.Genesis.Config
	if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(block.Number) == 0 {
		misc.ApplyDAOHardFork(stateDB)
	}
	gasPool := new(core.GasPool).AddGas(uint64(block.GasLimit))
	for i, tx := range block.Transactions {
		if this.FreeGas {
			tx_cpy := *tx
			tx_cpy.GasPrice = new(hexutil.Big)
			tx_cpy.Gas = ^hexutil.Uint64(0) / 100000
			tx = &tx_cpy
		}
		stateDB.Prepare(tx.Hash, block.Hash, i)
		txResult := this.BaseTrxEngine.ExecuteTransaction(&trx_engine_base.TransactionRequest{
			Transaction:      tx,
			BlockHeader:      &block.BlockHeader,
			DB:               stateDB,
			OnEvmInstruction: vm.NoopExecutionController,
			GasPool:          gasPool,
			CheckNonce:       !this.DisableNonceCheck,
		})
		var intermediateRoot []byte
		if chainConfig.IsByzantium(block.Number) {
			stateDB.Finalise(true)
		} else {
			intermediateRoot = stateDB.IntermediateRoot(chainConfig.IsEIP158(block.Number)).Bytes()
		}
		ret.UsedGas += hexutil.Uint64(txResult.GasUsed)
		ethReceipt := types.NewReceipt(intermediateRoot, txResult.ContractErr != nil, uint64(ret.UsedGas))
		if tx.To == nil {
			ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, uint64(tx.Nonce))
		}
		ethReceipt.TxHash = tx.Hash
		ethReceipt.GasUsed = txResult.GasUsed
		ethReceipt.Logs = stateDB.GetLogs(tx.Hash)
		ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
		ret.Receipts = append(ret.Receipts, ethReceipt)
		txErr := txResult.ConsensusErr
		if txErr == nil {
			txErr = txResult.ContractErr
		}
		util.Stringify(&txErr)
		ret.TransactionOutputs = append(ret.TransactionOutputs, &trx_engine.TransactionOutput{
			ReturnValue: txResult.EVMReturnValue,
			Error:       txErr,
		})
	}
	if !this.DisableMinerReward {
		var unclesMapped []*types.Header
		for _, uncle := range block.UncleBlocks {
			unclesMapped = append(unclesMapped, &types.Header{Number: uncle.Number.ToInt(), Coinbase: uncle.Miner})
		}
		ethash.AccumulateRewards(
			chainConfig,
			stateDB,
			&types.Header{Number: block.Number, Coinbase: block.Miner},
			unclesMapped)
	}
	if ret.StateRoot, err = stateDB.Commit(chainConfig.IsEIP158(block.Number)); err != nil {
		return
	}
	ret.TouchedExternallyOwnedAccountBalances = stateDB.TouchedExternallyOwnedAccountBalances
	return
}

func (this *EthTrxEngine) TransitionStateAndCommit(req *trx_engine.StateTransitionRequest) (ret *trx_engine.StateTransitionResult, err error) {
	ret, err = this.TransitionState(req)
	if err == nil {
		err = this.CommitToDisk(ret.StateRoot)
	}
	return
}
