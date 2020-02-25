package trx_engine_eth

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/big"
	"runtime"
	"runtime/debug"
	"testing"
	"time"
)

type BlockWithStateRoot = struct {
	*trx_engine.Block
	StateRoot common.Hash `json:"stateRoot"`
}

func Test_integration(t *testing.T) {
	block_db, err := (&rocksdb.Factory{
		File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain",
		ReadOnly:               true,
		Parallelism:            concurrent.CPU_COUNT,
		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
		MaxOpenFiles:           8192,
		OptimizeForPointLookup: 1024,
	}).NewInstance()
	util.PanicIfNotNil(err)
	getBlockByNumber := func(block_num uint64) *BlockWithStateRoot {
		key := []byte(fmt.Sprintf("%09d", block_num))
		block_json, err := block_db.Get(key)
		util.PanicIfNotNil(err)
		ret := new(BlockWithStateRoot)
		util.PanicIfNotNil(json.Unmarshal(block_json, ret))
		return ret
	}
	factory := new(EthTrxEngineFactory)
	//factory.ReadDBConfig = &trx_engine_base.StateDBConfig{
	//	DBFactory: &rocksdb.Factory{
	//		File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/state",
	//		ReadOnly:               true,
	//		Parallelism:            concurrent.CPU_COUNT,
	//		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
	//		OptimizeForPointLookup: 2 * 1024,
	//		UseDirectReads:         true,
	//	},
	//}
	//factory.WriteDBConfig = &trx_engine_base.StateDBConfig{DBFactory: new(memory.Factory)}
	//factory.ReadDBConfig = &trx_engine_base.StateDBConfig{DBFactory: new(memory.Factory)}
	factory.DBConfig = &trx_engine_base.StateDBConfig{DBFactory: &rocksdb.Factory{
		File:                   "/tmp/ololololo3",
		Parallelism:            concurrent.CPU_COUNT,
		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
		OptimizeForPointLookup: 3 * 1024,
		MaxOpenFiles:           8192,
	}}
	factory.BlockHashSourceFactory = trx_engine_base.SimpleBlockHashSourceFactory(func(blockNumber uint64) common.Hash {
		return getBlockByNumber(blockNumber).Hash
	})
	go func() {
		return
		measure_interval := 10 * time.Microsecond
		report_interval := 20 * time.Second
		time.Sleep(measure_interval)
		max := runtime.NumGoroutine()
		min := max
		sum := max
		count := 1
		last_report_time := time.Now()
		for {
			time.Sleep(measure_interval)
			num := runtime.NumGoroutine()
			if num < min {
				min = num
			} else if num > max {
				max = num
			}
			sum += num
			count++
			if now := time.Now(); now.Sub(last_report_time) > report_interval {
				fmt.Println("num goroutines: avg", float64(sum)/float64(count), "min", min, "max", max)
				last_report_time = now
			}
		}
	}()
	engine, cleanup, err := factory.NewInstance()
	util.PanicIfNotNil(err)
	defer cleanup()
	b, err := engine.DB.Get(binary.BytesView("last_block"))
	util.PanicIfNotNil(err)
	StartBlock := new(big.Int).SetBytes(b).Uint64()
	EndBlock := StartBlock + 10000
	var prevBlock *BlockWithStateRoot
	if StartBlock > 0 {
		prevBlock = getBlockByNumber(StartBlock - 1)
	}
	debug.SetGCPercent(-1)
	var max_heap_size uint64
	var mem_stats runtime.MemStats
	for blockNum := StartBlock; blockNum <= EndBlock; blockNum++ {
		block := getBlockByNumber(blockNum)
		fmt.Println("block", blockNum, "tx_count", len(block.Transactions))
		stateTransitionRequest := &trx_engine.StateTransitionRequest{Block: block.Block}
		if prevBlock != nil {
			stateTransitionRequest.BaseStateRoot = prevBlock.StateRoot
		}
		result, err := engine.TransitionState(stateTransitionRequest)
		util.PanicIfNotNil(err)
		util.Assert(result.StateRoot == block.StateRoot, result.StateRoot.Hex(), "!=", block.StateRoot.Hex())
		engine.DB.PutAsync(binary.BytesView("last_block"), new(big.Int).SetUint64(blockNum+1).Bytes())
		engine.DB.CommitAsync()
		prevBlock = block
		if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
			fmt.Println("gc")
			runtime.GC()
			max_heap_size = mem_stats.HeapAlloc * 3
		}
	}
}
