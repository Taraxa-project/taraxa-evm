package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa_vm/conflict_tracking"
	"math/big"
)

type TransactionData struct {
	To       *common.Address
	From     common.Address
	Nonce    uint64
	Amount   *big.Int
	GasLimit uint64
	GasPrice *big.Int
	Data     []byte
}

// TODO eliminate this as it's not needed by taraxa protocol???
type BlockData struct {
	Coinbase   common.Address
	Number     *big.Int
	Time       *big.Int
	Difficulty *big.Int
	GasLimit   uint64
	Hash       common.Hash
}

type StateTransition struct {
	StateRoot  common.Hash
	BlockData    *BlockData
	Transactions []*TransactionData
	//dbAddress    string
}

type Opts struct {
	getBlockHashByNumber func(uint64) common.Hash
}

type Result struct {
	StateRoot common.Hash
	Conflicts *conflict_tracking.Conflicts
	Receipts  types.Receipts
	AllLogs   []*types.Log
	UsedGas   uint64
}

func Process(persistentDB ethdb.Database, config *StateTransition, setOpts func(*Opts)) (result Result, err error) {
	opts := Opts{
		getBlockHashByNumber: func(u uint64) common.Hash {
			return common.BigToHash(big.NewInt(int64(u)))
		},
	}
	if setOpts != nil {
		setOpts(&opts)
	}
	result.Conflicts = new(conflict_tracking.Conflicts).Init()
	chainConfig := &params.ChainConfig{
		ChainID:             big.NewInt(0),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		Ethash:              new(params.EthashConfig),
	}
	evmConfig := vm.Config{}
	gasPool := new(core.GasPool).AddGas(config.BlockData.GasLimit);
	commonStateDB, err := state.New(config.StateRoot, state.NewDatabase(persistentDB))
	if err != nil {
		return
	}
	for ordinal, txData := range config.Transactions {
		txLocalDB := new(conflict_tracking.ConflictTrackingStateDB).Init(ordinal, commonStateDB, result.Conflicts)
		tx := types.NewMessage(
			txData.From, txData.To, txData.Nonce, txData.Amount,
			txData.GasLimit, txData.GasPrice, txData.Data, true,
		)
		txHash := types.RlpHash(tx);
		commonStateDB.Prepare(txHash, config.BlockData.Hash, ordinal)
		evmContext := vm.Context{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     opts.getBlockHashByNumber,
			Origin:      tx.From(),
			Coinbase:    config.BlockData.Coinbase,
			BlockNumber: new(big.Int).Set(config.BlockData.Number),
			Time:        new(big.Int).Set(config.BlockData.Time),
			Difficulty:  new(big.Int).Set(config.BlockData.Difficulty),
			GasLimit:    config.BlockData.GasLimit,
			GasPrice:    new(big.Int).Set(tx.GasPrice()),
		}
		vmenv := vm.NewEVM(evmContext, txLocalDB, chainConfig, evmConfig)
		var gas uint64
		_, gas, err = core.ApplyMessage(vmenv, tx, gasPool)
		if err != nil {
			return
		}
		result.UsedGas += gas
		intermediateRoot := commonStateDB.IntermediateRoot(true)
		receipt := types.NewReceipt(intermediateRoot.Bytes(), false, result.UsedGas)
		receipt.TxHash = txHash;
		receipt.GasUsed = gas
		if tx.To() == nil {
			receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
		}
		receipt.Logs = commonStateDB.GetLogs(txHash)
		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
		result.Receipts = append(result.Receipts, receipt)
		result.AllLogs = append(result.AllLogs, receipt.Logs...)
	}
	result.StateRoot, err = Flush(commonStateDB, func(opts *FlushOpts) {
		opts.report = true
	})
	return
}
