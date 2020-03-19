package trx_engine_eth

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type EthTrxEngine struct {
	*trx_engine_base.BaseTrxEngine
	EthTrxEngineConfig
}

func (self *EthTrxEngine) TransitionState(base_root common.Hash, blocks ...*trx_engine.Block) (ret *trx_engine.StateTransitionResult, err error) {
	ret = new(trx_engine.StateTransitionResult)
	defer util.Stringify(&err)
	stateDB := state.New(base_root, self.DB)
	chainConfig := self.Genesis.Config
	for _, block := range blocks {
		eip158 := chainConfig.IsEIP158(block.Number)
		if block.Number.Sign() == 0 {
			self.Genesis.Apply(stateDB)
			stateDB.Checkpoint(eip158)
			continue
		}
		if chainConfig.DAOForkSupport && chainConfig.DAOForkBlock != nil && chainConfig.DAOForkBlock.Cmp(block.Number) == 0 {
			misc.ApplyDAOHardFork(stateDB)
		}
		gasPool := new(core.GasPool).AddGas(uint64(block.GasLimit))
		for i, tx := range block.Transactions {
			if self.DisableGasFee {
				tx_cpy := *tx
				tx_cpy.GasPrice = new(hexutil.Big)
				tx_cpy.Gas = ^hexutil.Uint64(0) / 100000
				tx = &tx_cpy
			}
			stateDB.SetTransactionMetadata(tx.Hash, i)
			txResult := self.BaseTrxEngine.ExecuteTransaction(&trx_engine_base.TransactionRequest{
				Transaction:        tx,
				BlockHeader:        &block.BlockHeader,
				DB:                 stateDB,
				GasPool:            gasPool,
				CheckNonce:         !self.DisableNonceCheck,
				DisableMinerReward: self.DisableMinerReward,
			})
			txErr := txResult.ConsensusErr
			if txErr == nil {
				txErr = txResult.ContractErr
			}
			util.Stringify(&txErr)
			stateDB.Checkpoint(eip158)
			ret.UsedGas += hexutil.Uint64(txResult.GasUsed)
			ethReceipt := types.NewReceipt(nil, txErr != nil, uint64(ret.UsedGas))
			if tx.To == nil {
				ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, uint64(tx.Nonce))
			}
			ethReceipt.TxHash = tx.Hash
			ethReceipt.GasUsed = txResult.GasUsed
			ethReceipt.Logs = stateDB.GetLogs(tx.Hash)
			ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
			ret.Receipts = append(ret.Receipts, ethReceipt)
			ret.TransactionOutputs = append(ret.TransactionOutputs, &trx_engine.TransactionOutput{
				ReturnValue: txResult.EVMReturnValue,
				Error:       txErr,
			})
		}
		if !self.DisableMinerReward {
			unclesMapped := make([]*ethash.BlockNumAndCoinbase, len(block.UncleBlocks))
			for i, uncle := range block.UncleBlocks {
				unclesMapped[i] = &ethash.BlockNumAndCoinbase{Number: uncle.Number.ToInt(), Coinbase: uncle.Miner}
			}
			ethash.AccumulateRewards(
				chainConfig,
				stateDB,
				&ethash.BlockNumAndCoinbase{Number: block.Number, Coinbase: block.Miner},
				unclesMapped)
			stateDB.Checkpoint(eip158)
		}
	}
	ret.StateRoot = stateDB.Commit()
	return
}

func (self *EthTrxEngine) TransitionStateAndCommit(req *trx_engine.StateTransitionRequest) (ret *trx_engine.StateTransitionResult, err error) {
	//ret, err = self.TransitionState(req)
	//if err == nil {
	//	err = self.CommitToDisk()
	//}
	return
}
