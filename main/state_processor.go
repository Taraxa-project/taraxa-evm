package main

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
	"github.com/Taraxa-project/taraxa-evm/main/external"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

func Process(config *RunConfiguration) (result Result, err error) {
	defer util.RecoverErr(func(caught error) {
		result.Error = caught
		err = caught
	})
	ldbConfig := config.LDBConfig
	ldbDatabase, ldbErr := ethdb.NewLDBDatabase(ldbConfig.File, ldbConfig.Cache, ldbConfig.Handles)
	util.FailOnErr(ldbErr)
	defer ldbDatabase.Close()
	commonStateDB, stateDbErr := state.New(config.StateRoot, state.NewDatabase(ldbDatabase))
	util.FailOnErr(stateDbErr)
	gasPool := new(core.GasPool).AddGas(config.Block.GasLimit);
	evmConfig := vm.Config{}
	conflicts := new(conflict_tracking.Conflicts).Init()
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
	for ordinal, txData := range config.Transactions {
		txLocalDB := new(conflict_tracking.ConflictTrackingStateDB).Init(uint64(ordinal), commonStateDB, conflicts)
		tx := types.NewMessage(
			txData.From, txData.To, txData.Nonce, txData.Amount,
			txData.GasLimit, txData.GasPrice, txData.Data, true,
		)
		txHash := types.RlpHash(tx);
		commonStateDB.Prepare(txHash, config.Block.Hash, ordinal)
		evmContext := vm.Context{
			CanTransfer: core.CanTransfer,
			Transfer:    core.Transfer,
			GetHash:     external.GetHeaderHashByBlockNumber,
			Origin:      tx.From(),
			Coinbase:    config.Block.Coinbase,
			BlockNumber: new(big.Int).Set(config.Block.Number),
			Time:        new(big.Int).Set(config.Block.Time),
			Difficulty:  new(big.Int).Set(config.Block.Difficulty),
			GasLimit:    config.Block.GasLimit,
			GasPrice:    new(big.Int).Set(tx.GasPrice()),
		}
		vmenv := vm.NewEVM(evmContext, txLocalDB, chainConfig, evmConfig)
		returnValue, gas, txErr := core.ApplyMessage(vmenv, tx, gasPool)
		util.FailOnErr(txErr)
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
		result.ReturnValues = append(result.ReturnValues, returnValue)
	}
	result.ConcurrentSchedule = &ConcurrentSchedule{
		Sequential: conflicts.GetConflictingTransactions(),
	}
	finalRoot, flushErr := Flush(commonStateDB, nil)
	util.FailOnErr(flushErr)
	result.StateRoot = finalRoot

	// TODO remove
	in, err := json.Marshal(config)
	util.FailOnErr(err)
	fmt.Println("IN: " + string(in))

	// TODO remove
	out, err := json.Marshal(result)
	util.FailOnErr(err)
	fmt.Println("OUT: " + string(out))

	return
}
