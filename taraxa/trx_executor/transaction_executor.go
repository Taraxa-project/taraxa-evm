package trx_executor

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
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type TransactionExecutor struct {
	DB                 *state.Database
	GetBlockHash       vm.GetHashFunc
	Genesis            *core.Genesis
	EvmConfig          vm.Config
	DisableMinerReward bool
	DisableNonceCheck  bool
	DisableGasFee      bool
}

func (self *TransactionExecutor) ExecBlocks(base_root common.Hash, blocks ...*Block) *ExecutionResult {
	ret := new(ExecutionResult)
	state_db := state.New(base_root, self.DB)
	chain_cfg := self.Genesis.Config
	for _, block := range blocks {
		eip158 := chain_cfg.IsEIP158(block.Number)
		if block.Number.Sign() == 0 {
			self.Genesis.Apply(state_db)
			state_db.Checkpoint(eip158)
			continue
		}
		if chain_cfg.DAOForkSupport && chain_cfg.DAOForkBlock != nil && chain_cfg.DAOForkBlock.Cmp(block.Number) == 0 {
			misc.ApplyDAOHardFork(state_db)
		}
		gas_pool := new(core.GasPool).AddGas(uint64(block.GasLimit))
		for i, tx := range block.Transactions {
			if self.DisableGasFee {
				tx_cpy := *tx
				tx_cpy.GasPrice = new(hexutil.Big)
				tx_cpy.Gas = ^hexutil.Uint64(0) / 100000
				tx = &tx_cpy
			}
			state_db.SetTransactionMetadata(tx.Hash, i)
			txResult := self.ExecuteTransaction(&TransactionRequest{
				Transaction:        tx,
				BlockHeader:        &block.BlockHeader,
				DB:                 state_db,
				GasPool:            gas_pool,
				CheckNonce:         !self.DisableNonceCheck,
				DisableMinerReward: self.DisableMinerReward,
			})
			txErr := txResult.ConsensusErr
			if txErr == nil {
				txErr = txResult.ContractErr
			}
			util.Stringify(&txErr)
			state_db.Checkpoint(eip158)
			ret.UsedGas += hexutil.Uint64(txResult.GasUsed)
			ethReceipt := types.NewReceipt(nil, txErr != nil, uint64(ret.UsedGas))
			if tx.To == nil {
				ethReceipt.ContractAddress = crypto.CreateAddress(tx.From, uint64(tx.Nonce))
			}
			ethReceipt.TxHash = tx.Hash
			ethReceipt.GasUsed = txResult.GasUsed
			ethReceipt.Logs = state_db.GetLogs(tx.Hash)
			ethReceipt.Bloom = types.CreateBloom(types.Receipts{ethReceipt})
			ret.Receipts = append(ret.Receipts, ethReceipt)
			ret.TransactionOutputs = append(ret.TransactionOutputs, &TransactionOutput{
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
				chain_cfg,
				state_db,
				&ethash.BlockNumAndCoinbase{Number: block.Number, Coinbase: block.Miner},
				unclesMapped)
			state_db.Checkpoint(eip158)
		}
	}
	ret.StateRoot = state_db.Commit()
	return ret
}

func (self *TransactionExecutor) ExecuteTransaction(req *TransactionRequest) *TransactionResult {
	msg := types.NewMessage(
		req.Transaction.From, req.Transaction.To, uint64(req.Transaction.Nonce),
		req.Transaction.Value.ToInt(), uint64(req.Transaction.Gas),
		req.Transaction.GasPrice.ToInt(), req.Transaction.Input, req.CheckNonce)
	evmContext := vm.Context{
		GetHash:     self.GetBlockHash,
		Origin:      msg.From(),
		Coinbase:    req.BlockHeader.Miner,
		BlockNumber: req.BlockHeader.Number,
		Time:        req.BlockHeader.Time.ToInt(),
		Difficulty:  req.BlockHeader.Difficulty.ToInt(),
		GasLimit:    uint64(req.BlockHeader.GasLimit),
		GasPrice:    msg.GasPrice(),
	}
	evm := vm.NewEVM(evmContext, req.DB, self.Genesis.Config, &self.EvmConfig)
	ret, usedGas, vmErr, consensusErr := core.
		NewStateTransition(evm, msg, req.GasPool, req.DisableMinerReward).
		TransitionDb()
	return &TransactionResult{ret, usedGas, vmErr, consensusErr}
}

type TransactionRequest = struct {
	Transaction        *Transaction
	BlockHeader        *BlockHeader
	GasPool            *core.GasPool
	DB                 vm.StateDB
	CheckNonce         bool
	DisableMinerReward bool
}

type TransactionResult = struct {
	EVMReturnValue []byte
	GasUsed        uint64
	ContractErr    error
	ConsensusErr   error
}
