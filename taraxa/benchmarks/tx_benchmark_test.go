package benchmarks

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/math"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/benchmarking"
	"math/big"
	"os"
	"testing"
)

func BenchmarkRoot(b *testing.B) {
	test_amount := new(big.Int).SetUint64(1000)
	sender := common.BigToAddress(new(big.Int).SetUint64(100))
	receiver := common.BigToAddress(new(big.Int).SetUint64(1001))
	evm_cfg := &vm.Config{StaticConfig: new(vm.StaticConfig)}
	gas_limit := uint64(math.MaxUint64)
	coinbase := common.Address{}
	tx_hash, block_hash := common.Hash{}, common.Hash{}
	db_path := os.TempDir() + string(os.PathSeparator) + "tx_bench"
	benchmarking.AddBenchmark(b, "single_coin_tx_no_cache", func(b *testing.B, i int) {
		b.StopTimer()
		raw_db, err0 := (&rocksdb.Factory{
			File:           db_path,
			UseDirectReads: true,
		}).NewInstance()
		util.PanicIfNotNil(err0)
		defer util.PanicIfNotNil(os.RemoveAll(db_path))
		defer raw_db.Close()
		base_root := func() common.Hash {
			db := state.NewDatabase(raw_db)
			state_db, err := state.New(common.Hash{}, db)
			util.PanicIfNotNil(err)
			state_db.SetBalance(sender, test_amount)
			state_db.CreateAccount(receiver)
			base_root, err2 := state_db.Commit(false)
			util.PanicIfNotNil(err2)
			err3 := db.TrieDB().Commit(base_root, false, nil)
			util.PanicIfNotNil(err3)
			return base_root
		}()
		db := state.NewDatabaseWithCache(raw_db, 1024)
		state_db, err := state.New(base_root, db)
		util.PanicIfNotNil(err)
		gas_pool := new(core.GasPool).AddGas(gas_limit)
		b.StartTimer()

		evm_ctx := vm.Context{
			GetHash:     nil,
			Origin:      sender,
			Coinbase:    coinbase,
			BlockNumber: common.Big0,
			Time:        common.Big0,
			Difficulty:  common.Big0,
			GasLimit:    gas_limit,
			GasPrice:    common.Big0,
		}
		evm := vm.NewEVM(evm_ctx, state_db, params.MainnetChainConfig, evm_cfg)
		msg := types.NewMessage(
			evm_ctx.Origin, &receiver, 0, test_amount,
			evm_ctx.GasLimit, evm_ctx.GasPrice, nil, true)
		state_transition := core.NewStateTransition(evm, msg, gas_pool)
		state_db.Prepare(tx_hash, block_hash, 0)
		_, usedGas, vmErr, consensusErr := state_transition.TransitionDb()
		root, err43 := state_db.Commit(true)
		receipt := types.NewReceipt(root.Bytes(), false, usedGas+1)
		receipt.TxHash = tx_hash
		receipt.Logs = state_db.GetLogs(tx_hash)
		receipt.Bloom = types.BytesToBloom(types.LogsBloom(receipt.Logs).Bytes())

		b.StopTimer()
		util.PanicIfNotNil(err43)
		util.PanicIfNotNil(vmErr)
		util.PanicIfNotNil(consensusErr)
	})
	benchmarking.AddBenchmark(b, "single_coin_tx_with_cache", func(b *testing.B, i int) {
		b.StopTimer()
		raw_db, err0 := (&rocksdb.Factory{
			File:           db_path,
			UseDirectReads: true,
		}).NewInstance()
		util.PanicIfNotNil(err0)
		defer util.PanicIfNotNil(os.RemoveAll(db_path))
		defer raw_db.Close()
		db := state.NewDatabaseWithCache(raw_db, 1024)
		state_db, err := state.New(common.Hash{}, db)
		util.PanicIfNotNil(err)
		state_db.SetBalance(sender, test_amount)
		state_db.CreateAccount(receiver)
		gas_pool := new(core.GasPool).AddGas(gas_limit)
		b.StartTimer()

		evm_ctx := vm.Context{
			GetHash:     nil,
			Origin:      sender,
			Coinbase:    coinbase,
			BlockNumber: common.Big0,
			Time:        common.Big0,
			Difficulty:  common.Big0,
			GasLimit:    gas_limit,
			GasPrice:    common.Big0,
		}
		evm := vm.NewEVM(evm_ctx, state_db, params.MainnetChainConfig, evm_cfg)
		msg := types.NewMessage(
			evm_ctx.Origin, &receiver, 0, test_amount,
			evm_ctx.GasLimit, evm_ctx.GasPrice, nil, true)
		state_transition := core.NewStateTransition(evm, msg, gas_pool)
		state_db.Prepare(tx_hash, block_hash, 0)
		_, usedGas, vmErr, consensusErr := state_transition.TransitionDb()
		root, err43 := state_db.Commit(true)
		state_db.Finalise(true)
		receipt := types.NewReceipt(root[:], false, usedGas+1)
		receipt.TxHash = tx_hash
		receipt.Logs = state_db.GetLogs(tx_hash)
		receipt.Bloom = types.BytesToBloom(types.LogsBloom(receipt.Logs).Bytes())

		b.StopTimer()
		util.PanicIfNotNil(err43)
		util.PanicIfNotNil(vmErr)
		util.PanicIfNotNil(consensusErr)
	})
}
