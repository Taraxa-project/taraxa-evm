package main

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_eth"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math"
	"math/big"
	//"net/http"
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"
	//"runtime/pprof"
	"time"
)

type BlockWithStateRoot = struct {
	*trx_engine.Block
	StateRoot common.Hash `json:"stateRoot"`
}

func main() {
	//go func() {
	//	util.PanicIfNotNil(http.ListenAndServe("localhost:6060", nil))
	//}()
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
	factory := new(trx_engine_eth.EthTrxEngineFactory)
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
		OptimizeForPointLookup: 4 * 1024,
		MaxOpenFiles:           7000,
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
	start_block_num := uint64(0)
	if b != nil {
		start_block_num = new(big.Int).SetBytes(b).Uint64() + 1
	}
	end_block_num := start_block_num + 30000000

	blocks := make(chan *BlockWithStateRoot, 64)
	block_load_requests := make(chan uint32, 32)
	defer close(block_load_requests)
	go func() {
		defer close(blocks)
		if start_block_num == 0 {
			blocks <- nil
		} else {
			blocks <- getBlockByNumber(start_block_num - 1)
		}
		next_to_load := start_block_num
		for {
			to_load_count, ok := <-block_load_requests
			if !ok {
				break
			}
			for i := uint32(0); i < to_load_count; i++ {
				blocks <- getBlockByNumber(next_to_load)
				next_to_load++
			}
		}
	}()
	block_load_requests <- 15
	last_block := <-blocks

	//profile_basedir := "/Users/compuktor/projects/taraxa.io/taraxa-evm/taraxa/trx_engine/trx_engine_eth/main/profiles/"
	//util.PanicIfNotNil(os.MkdirAll(profile_basedir, os.ModePerm))
	//new_prof_file := func(time time.Time, kind string) *os.File {
	//	ret, err := os.Create(profile_basedir + strconv.FormatInt(time.Unix(), 10) + "_" + kind + ".prof")
	//	util.PanicIfNotNil(err)
	//	return ret
	//}
	//last_profile_snapshot_time := time.Now()
	//write_reset_profiles := func(start_time time.Time) {
	//	fmt.Println("writing profiles...")
	//	pprof.StopCPUProfile()
	//	for _, prof := range pprof.Profiles() {
	//		prof.WriteTo(new_prof_file(last_profile_snapshot_time, prof.Name()), 1)
	//	}
	//	last_profile_snapshot_time = start_time
	//	pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu"))
	//}
	//pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu"))

	debug.SetGCPercent(-1)
	var max_heap_size uint64
	var mem_stats runtime.MemStats

	block_buf := make([]*trx_engine.Block, 0, 32)
	tps_sum := 0.0
	tps_cnt := 0
	tps_min := math.MaxFloat64
	tps_max := -1.0

	const min_tx_to_execute = 5000
	for blockNum := start_block_num; blockNum <= end_block_num; {
		var base_root common.Hash
		if last_block != nil {
			base_root = last_block.StateRoot
		}
		tx_count := 0
		for ; tx_count < min_tx_to_execute; blockNum++ {
			block_load_requests <- 1
			last_block = <-blocks
			tx_count += len(last_block.Transactions)
			block_buf = append(block_buf, last_block.Block)
		}
		fmt.Println("blocks:", int(blockNum)-len(block_buf), "-", blockNum-1, "tx_count:", tx_count)
		now := time.Now()
		result, err := engine.TransitionState(base_root, block_buf...)
		tps := float64(tx_count) / time.Now().Sub(now).Seconds()
		tps_sum += tps
		tps_cnt++
		if tps < tps_min {
			tps_min = tps
		}
		if tps_max < tps {
			tps_max = tps
		}
		fmt.Println("TPS current:", tps, "avg:", tps_sum/float64(tps_cnt), "min:", tps_min, "max:", tps_max)
		block_buf = block_buf[:0]
		util.PanicIfNotNil(err)
		util.Assert(result.StateRoot == last_block.StateRoot, result.StateRoot.Hex(), "!=", last_block.StateRoot.Hex())
		engine.DB.PutAsync(binary.BytesView("last_block"), last_block.Number.Bytes())
		engine.DB.CommitAsync()
		if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
			//write_reset_profiles(time.Now())
			fmt.Println("gc...")
			runtime.GC()
			runtime.ReadMemStats(&mem_stats)
			max_heap_size = mem_stats.HeapAlloc * 4
		}
	}
	engine.DB.Join()
}
