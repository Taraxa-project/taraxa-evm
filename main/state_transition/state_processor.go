package state_transition
//
//import (
//	"github.com/Taraxa-project/taraxa-evm/common"
//	"github.com/Taraxa-project/taraxa-evm/core"
//	"github.com/Taraxa-project/taraxa-evm/core/state"
//	"github.com/Taraxa-project/taraxa-evm/core/types"
//	"github.com/Taraxa-project/taraxa-evm/core/vm"
//	"github.com/Taraxa-project/taraxa-evm/crypto"
//	"github.com/Taraxa-project/taraxa-evm/ethdb"
//	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
//	"github.com/Taraxa-project/taraxa-evm/main/util"
//	"github.com/Taraxa-project/taraxa-evm/params"
//	"github.com/emirpasic/gods/sets/hashset"
//	"math/big"
//)
//
//func Run(config *RunConfiguration, externalApi *ExternalApi) (result Result, err error) {
//	defer util.Catch(func(caught error) {
//		err = caught
//		result.Error = err
//	})
//	ldbConfig := config.LDBConfig
//	ldbDatabase, ldbErr := ethdb.NewLDBDatabase(ldbConfig.File, ldbConfig.Cache, ldbConfig.Handles)
//	util.PanicOn(ldbErr)
//	defer ldbDatabase.Close()
//
//	gasPool := new(core.GasPool).AddGas(config.Block.GasLimit);
//	evmConfig := new(vm.Config)
//	conflicts := new(conflict_tracking.ConflictDetector).Init()
//	chainConfig := &params.ChainConfig{
//		ChainID:             big.NewInt(0),
//		HomesteadBlock:      big.NewInt(0),
//		EIP150Block:         big.NewInt(0),
//		EIP155Block:         big.NewInt(0),
//		EIP158Block:         big.NewInt(0),
//		ByzantiumBlock:      big.NewInt(0),
//		ConstantinopleBlock: big.NewInt(0),
//		PetersburgBlock:     big.NewInt(0),
//		Ethash:              new(params.EthashConfig),
//	}
//	blockNumber := BigInt(config.Block.Number)
//	trieDB := state.NewDatabase(ldbDatabase)
//
//	txContexts := make([]*TransactionParams, len(config.Transactions))
//
//	sequentialTx := hashset.New(config.ConcurrentSchedule.Sequential...)
//
//	go func() {
//		// TODO sorting
//		for _, ordinal := range config.ConcurrentSchedule.Sequential {
//
//		}
//	}()
//	for ordinal, txData := range config.Transactions {
//		txContexts[ordinal] =
//			commonStateDB, stateDbErr := state.New(config.StateRoot, trieDB)
//		util.PanicOn(stateDbErr)
//
//		txExecution := TransactionExecution{
//			txId:      ordinal,
//			txHash:    txData.Hash,
//			blockHash: config.Block.Hash,
//			tx: types.NewMessage(
//				txData.From, txData.To, txData.Nonce, BigInt(txData.Amount),
//				txData.GasLimit, BigInt(txData.GasPrice), *txData.Data,
//				true,
//			),
//			chainConfig: chainConfig,
//			evmContext: vm.Context{
//				CanTransfer: core.CanTransfer,
//				Transfer:    core.Transfer,
//				GetHash:     externalApi.GetHeaderHashByBlockNumber,
//				Origin:      tx.From(),
//				Coinbase:    config.Block.Coinbase,
//				BlockNumber: blockNumber,
//				Time:        BigInt(config.Block.Time),
//				Difficulty:  BigInt(config.Block.Difficulty),
//				GasLimit:    config.Block.GasLimit,
//				GasPrice:    new(big.Int).Set(tx.GasPrice()),
//			},
//			evmConfig: evmConfig,
//		}
//
//		go txExecution.Run(commonStateDB, &TransactionParams{
//			stateDB:   state.New(config.StateRoot, state.NewDatabase(ldbDatabase)),
//			conflicts: conflicts,
//			gasPool:   gasPool,
//		})
//
//		result.UsedGas += gas
//
//		var intermediateRootBytes []byte
//		if chainConfig.IsByzantium(header.Number) {
//			commonStateDB.Finalise(true)
//		} else {
//			intermediateRootBytes = commonStateDB.IntermediateRoot(chainConfig.IsEIP158(blockNumber)).Bytes()
//		}
//		receipt := types.NewReceipt(intermediateRootBytes, false, result.UsedGas)
//		receipt.TxHash = txHash;
//		receipt.GasUsed = gas
//		if tx.To() == nil {
//			receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
//		}
//		receipt.Logs = commonStateDB.GetLogs(txHash)
//		receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
//		result.Receipts = append(result.Receipts, receipt)
//		result.AllLogs = append(result.AllLogs, receipt.Logs...)
//		result.ReturnValues = append(result.ReturnValues, &returnValue)
//	}
//
//	var commitContext *struct {
//		currentRoot common.Hash
//		db state.Database
//	}
//
//	for _, txContext := range txContexts {
//		txResult := <-txContext.result
//		if commitContext != nil {
//			txContext.stateDB.Rebase(commitContext.currentRoot, commitContext.db)
//		}
//		txContext.stateDB.Finalise(true)
//
//		txContext.stateDB.Commit()
//		txContext.stateDB
//	}
//
//	result.ConcurrentSchedule = &ConcurrentSchedule{
//		Sequential: conflicts.Reset(),
//	}
//	finalRoot, flushErr := Flush(commonStateDB, func(opts *FlushOpts) {
//		opts.deleteEmptyObjects = chainConfig.IsEIP158(blockNumber)
//	})
//	util.PanicOn(flushErr)
//	result.StateRoot = finalRoot
//	return
//}
//
//func GenerateSchedule(configuration *RunConfiguration, api *ExternalApi) {
//
//}
//
//func MakeStateTransition(configuration *RunConfiguration, api *ExternalApi) {
//	util.AssertNotNil(configuration.ConcurrentSchedule)
//
//}
//
//func executeTx(message types.Message) {
//
//}
