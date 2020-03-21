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
	_ "net/http/pprof"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"time"
)

func main() {
	usr_dir, err := os.UserHomeDir()
	util.PanicIfNotNil(err)
	dest_data_dir := mkdirp(usr_dir + "/taraxa_evm_data")

	debug.SetGCPercent(-1)
	var max_heap_size uint64
	var mem_stats runtime.MemStats

	//go func() {
	//	util.PanicIfNotNil(http.ListenAndServe("localhost:6060", nil))
	//}()
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
	//profile_basedir := mkdirp(dest_data_dir + "/profiles")
	//util.PanicIfNotNil(exec.Command("mkdir", "-p", profile_basedir).Run())
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

	rocksdb := rocksdb.New(&rocksdb.Config{
		File:                   mkdirp(dest_data_dir + "/foo"),
		Parallelism:            runtime.NumCPU(),
		MaxFileOpeningThreads:  runtime.NumCPU(),
		OptimizeForPointLookup: 4 * 1024,
		MaxOpenFiles:           7000,
	})
	defer rocksdb.Close()
	db := state.NewDatabase(rocksdb)
	//db.ToggleMemOnly()
	engine := &trx_executor.TransactionExecutor{
		DB: db,
		GetBlockHash: func(blockNumber uint64) common.Hash {
			return getBlockByNumber(blockNumber).Hash
		},
		Genesis: core.DefaultGenesisBlock(),
	}

	min_tx_to_execute := 10000
	blocks := make(chan *BlockInfo, min_tx_to_execute/10)
	block_load_requests := make(chan interface{}, cap(blocks))
	defer close(block_load_requests)
	go func() {
		defer block_db.Close()
		defer close(blocks)
		var next_to_load uint64
		if last_block_num_b := db.GetCommitted(binary.BytesView("last_block")); last_block_num_b != nil {
			next_to_load = new(big.Int).SetBytes(last_block_num_b).Uint64()
			blocks <- getBlockByNumber(next_to_load)
			next_to_load++
		} else {
			blocks <- nil
		}
		for i := 1; i < cap(blocks); i++ {
			blocks <- getBlockByNumber(next_to_load)
			next_to_load++
		}
		for {
			_, ok := <-block_load_requests
			if !ok {
				break
			}
			blocks <- getBlockByNumber(next_to_load)
			next_to_load++
		}
	}()

	tps_sum, tps_cnt, tps_min, tps_max := 0.0, 0, math.MaxFloat64, -1.0

	block_buf := make([]*trx_executor.Block, 0, 1<<8)
	last_block := <-blocks
	for {
		var base_root common.Hash
		if last_block != nil {
			base_root = last_block.StateRoot
		}
		tx_count := 0
		for {
			block_load_requests <- nil
			last_block = <-blocks
			tx_count += len(last_block.Transactions)
			if last_block.Number.Sign() == 0 {
				tx_count = 1
			}
			block_buf = append(block_buf, &last_block.Block)
			if tx_count >= min_tx_to_execute {
				break
			}
		}
		fmt.Println(
			"blocks:", block_buf[0].Number.String(), "-", block_buf[len(block_buf)-1].Number.String(),
			"tx_count:", tx_count)
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
		db.PutAsync(binary.BytesView("last_block"), last_block.Number.Bytes())
		db.Commit()
		if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
			//write_reset_profiles(time.Now())
			fmt.Println("gc...")
			runtime.GC()
			runtime.ReadMemStats(&mem_stats)
			max_heap_size = mem_stats.HeapAlloc * 4
		}
	}
}

func mkdirp(path string) string {
	util.PanicIfNotNil(exec.Command("mkdir", "-p", path).Run())
	return path
}
