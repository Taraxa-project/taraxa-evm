package main

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_executor"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"math"
	"math/big"
	"os"
	"os/exec"

	//"net/http"
	_ "net/http/pprof"
	"runtime"
	"runtime/debug"
	//"runtime/pprof"
	"time"
)

func main() {
	//go func() {
	//	util.PanicIfNotNil(http.ListenAndServe("localhost:6060", nil))
	//}()
	block_db := rocksdb.New(&rocksdb.Config{
		File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain",
		ReadOnly:               true,
		Parallelism:            runtime.NumCPU(),
		MaxFileOpeningThreads:  runtime.NumCPU(),
		MaxOpenFiles:           8192,
		OptimizeForPointLookup: 1024,
	})
	type BlockInfo = struct {
		Hash      common.Hash `json:"hash"`
		StateRoot common.Hash `json:"stateRoot"`
		trx_executor.Block
	}
	getBlockByNumber := func(block_num uint64) *BlockInfo {
		key := []byte(fmt.Sprintf("%09d", block_num))
		block_json, err := block_db.Get(key)
		util.PanicIfNotNil(err)
		ret := new(BlockInfo)
		util.PanicIfNotNil(json.Unmarshal(block_json, ret))
		return ret
	}
	usr_dir, err := os.UserHomeDir()
	util.PanicIfNotNil(err)
	statedb_dir := usr_dir + "/taraxa_evm_data/foo"
	util.PanicIfNotNil(exec.Command("mkdir", "-p", statedb_dir).Run())
	//go func() {
	//	return
	//	measure_interval := 10 * time.Microsecond
	//	report_interval := 20 * time.Second
	//	time.Sleep(measure_interval)
	//	max := runtime.NumGoroutine()
	//	min := max
	//	sum := max
	//	count := 1
	//	last_report_time := time.Now()
	//	for {
	//		time.Sleep(measure_interval)
	//		num := runtime.NumGoroutine()
	//		if num < min {
	//			min = num
	//		} else if num > max {
	//			max = num
	//		}
	//		sum += num
	//		count++
	//		if now := time.Now(); now.Sub(last_report_time) > report_interval {
	//			fmt.Println("num goroutines: avg", float64(sum)/float64(count), "min", min, "max", max)
	//			last_report_time = now
	//		}
	//	}
	//}()
	engine := &trx_executor.TransactionExecutor{
		DB: state.NewDatabase(rocksdb.New(&rocksdb.Config{
			File:                   statedb_dir,
			Parallelism:            runtime.NumCPU(),
			MaxFileOpeningThreads:  runtime.NumCPU(),
			OptimizeForPointLookup: 4 * 1024,
			MaxOpenFiles:           7000,
		})),
		GetBlockHash: func(blockNumber uint64) common.Hash {
			return getBlockByNumber(blockNumber).Hash
		},
		Genesis: core.DefaultGenesisBlock(),
	}
	b := engine.DB.GetCommitted(binary.BytesView("last_block"))
	start_block_num := uint64(0)
	if b != nil {
		start_block_num = new(big.Int).SetBytes(b).Uint64() + 1
	}
	end_block_num := start_block_num + 30000000

	blocks := make(chan *BlockInfo, 64)
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

	block_buf := make([]*trx_executor.Block, 0, 32)
	tps_sum := 0.0
	tps_cnt := 0
	tps_min := math.MaxFloat64
	tps_max := -1.0

	min_tx_to_execute := 1
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
			if last_block.Number.Sign() == 0 {
				tx_count = 1
			}
			block_buf = append(block_buf, &last_block.Block)
		}
		fmt.Println("blocks:", int(blockNum)-len(block_buf), "-", blockNum-1, "tx_count:", tx_count)
		now := time.Now()
		result := engine.ExecBlocks(base_root, block_buf...)
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
		util.Assert(result.StateRoot == last_block.StateRoot, result.StateRoot.Hex(), "!=", last_block.StateRoot.Hex())
		//break
		engine.DB.PutAsync(binary.BytesView("last_block"), last_block.Number.Bytes())
		engine.DB.Commit()
		if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
			//write_reset_profiles(time.Now())
			fmt.Println("gc...")
			runtime.GC()
			runtime.ReadMemStats(&mem_stats)
			max_heap_size = mem_stats.HeapAlloc * 4
		}
	}
}
