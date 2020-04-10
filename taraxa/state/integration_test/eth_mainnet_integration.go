package main

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
	"math"
	"math/big"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"
	"unsafe"
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
	profiling := false
	profile_basedir := mkdirp(dest_data_dir + "/profiles/")
	util.PanicIfNotNil(exec.Command("mkdir", "-p", profile_basedir).Run())
	util.PanicIfNotNil(os.MkdirAll(profile_basedir, os.ModePerm))
	new_prof_file := func(time time.Time, kind string) *os.File {
		ret, err := os.Create(profile_basedir + strconv.FormatInt(time.Unix(), 10) + "_" + kind + ".prof")
		util.PanicIfNotNil(err)
		return ret
	}
	last_profile_snapshot_time := time.Now()
	if profiling {
		pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu"))
	}
	write_reset_profiles := func(start_time time.Time) {
		if !profiling {
			return
		}
		fmt.Println("writing profiles...")
		pprof.StopCPUProfile()
		for _, prof := range pprof.Profiles() {
			prof.WriteTo(new_prof_file(last_profile_snapshot_time, prof.Name()), 1)
		}
		last_profile_snapshot_time = start_time
		pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu"))
	}

	opts_w_default := gorocksdb.NewDefaultWriteOptions()
	opts_r_default := gorocksdb.NewDefaultReadOptions()
	blk_db_opts := gorocksdb.NewDefaultOptions()
	blk_db_opts.SetErrorIfExists(false)
	blk_db_opts.SetCreateIfMissing(true)
	blk_db_opts.SetCreateIfMissingColumnFamilies(true)
	blk_db_opts.IncreaseParallelism(runtime.NumCPU())
	blk_db_opts.SetMaxFileOpeningThreads(runtime.NumCPU())
	blk_db_opts.SetMaxOpenFiles(8192)
	blk_db_opts.OptimizeForPointLookup(1024)
	blk_db, e0 := gorocksdb.OpenDbForReadOnly(
		blk_db_opts,
		"/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain",
		false)
	util.PanicIfNotNil(e0)
	defer blk_db.Close()

	type Transaction = struct {
		From     common.Address  `json:"from" gencodec:"required"`
		GasPrice *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		To       *common.Address `json:"to,omitempty"`
		Nonce    hexutil.Uint64  `json:"nonce" gencodec:"required"`
		Value    *hexutil.Big    `json:"value" gencodec:"required"`
		Gas      hexutil.Uint64  `json:"gas" gencodec:"required"`
		Input    hexutil.Bytes   `json:"input" gencodec:"required"`
	}
	type UncleBlock = struct {
		Number hexutil.Uint64 `json:"number"  gencodec:"required"`
		Miner  common.Address `json:"miner"  gencodec:"required"`
	}
	type VmBlock = struct {
		Number     types.BlockNum `json:"number" gencodec:"required"`
		Miner      common.Address `json:"miner" gencodec:"required"`
		GasLimit   hexutil.Uint64 `json:"gasLimit"  gencodec:"required"`
		Time       hexutil.Uint64 `json:"timestamp"  gencodec:"required"`
		Difficulty *hexutil.Big   `json:"difficulty"  gencodec:"required"`
	}
	type BlockInfo = struct {
		VmBlock
		UncleBlocks  []UncleBlock  `json:"uncleBlocks"  gencodec:"required"`
		Transactions []Transaction `json:"transactions"  gencodec:"required"`
		Hash         common.Hash   `json:"hash" gencodec:"required"`
		StateRoot    common.Hash   `json:"stateRoot" gencodec:"required"`
	}

	getBlockByNumber := func(block_num types.BlockNum) *BlockInfo {
		key := []byte(fmt.Sprintf("%09d", block_num))
		block_json, err := blk_db.GetBytes(opts_r_default, key)
		util.PanicIfNotNil(err)
		ret := new(BlockInfo)
		util.PanicIfNotNil(json.Unmarshal(block_json, ret))
		return ret
	}

	db_opts := gorocksdb.NewDefaultOptions()
	db_opts.SetErrorIfExists(false)
	db_opts.SetCreateIfMissing(true)
	db_opts.SetCreateIfMissingColumnFamilies(true)
	db_opts.IncreaseParallelism(runtime.NumCPU())
	db_opts.SetMaxFileOpeningThreads(runtime.NumCPU())
	db_opts.SetMaxOpenFiles(7000)
	const col_cnt = 6
	cfnames, cfopts := [col_cnt]string{}, [col_cnt]*gorocksdb.Options{}
	for i := 0; i < col_cnt; i++ {
		if i == 0 {
			cfnames[i] = "default"
		} else {
			cfnames[i] = strconv.Itoa(i)
		}
		cfopts[i] = gorocksdb.NewDefaultOptions()
	}
	db, cols, e1 := gorocksdb.OpenDbColumnFamilies(db_opts, mkdirp(dest_data_dir+"/foo"), cfnames[:], cfopts[:])
	util.PanicIfNotNil(e1)
	defer db.Close()

	var last_blk_num types.BlockNum
	var last_root *common.Hash
	last_block_num_b, err := db.GetBytes(opts_r_default, bin.BytesView("last_block"))
	util.PanicIfNotNil(err)
	if len(last_block_num_b) != 0 {
		last_blk_num = bin.DEC_b_endian_64(last_block_num_b)
		last_root = &getBlockByNumber(last_blk_num).StateRoot
	}
	var state_db state_rocksdb.RocksDBStateDB
	state_db.I(db, state_rocksdb.Columns{cols[1], cols[2], cols[3], cols[4], cols[5]})
	state_api := new(state.API).I(
		&state_db,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(getBlockByNumber(num).Hash[:])
		},
		*params.MainnetChainConfig,
		vm.ExecutionOptions{},
		false,
		last_blk_num,
		last_root,
		trie.TrieWriterOpts{},
	)
	state_transition_service := state_api.GetStateTransitionService()
	if last_root == nil {
		batch := gorocksdb.NewWriteBatch()
		state_db.BatchBegin(batch)
		root := state_transition_service.ApplyGenesis(core.MainnetGenesis().Alloc)
		assert.EQ(root.Hex(), getBlockByNumber(0).StateRoot.Hex())
		batch.Put(bin.BytesView("last_block"), bin.ENC_b_endian_64(0))
		state_db.BatchDone()
		util.PanicIfNotNil(db.Write(opts_w_default, batch))
	}

	min_tx_to_execute := 0
	blocks := make(chan *BlockInfo, min_tx_to_execute/10+3)
	block_load_requests := make(chan byte, cap(blocks))
	defer close(block_load_requests)
	go func() {
		defer close(blocks)
		next_to_load := last_blk_num + 1
		for i := 0; i < cap(blocks); i++ {
			blocks <- getBlockByNumber(next_to_load)
			next_to_load++
		}
		for {
			if _, ok := <-block_load_requests; !ok {
				break
			}
			blocks <- getBlockByNumber(next_to_load)
			next_to_load++
		}
	}()

	tps_sum, tps_cnt, tps_min, tps_max := 0.0, 0, math.MaxFloat64, -1.0
	block_buf := make([]*BlockInfo, 0, 1<<8)
	for {
		tx_count := 0
		for {
			block_load_requests <- 0
			last_block := <-blocks
			block_buf = append(block_buf, last_block)
			tx_count += len(last_block.Transactions)
			if tx_count >= min_tx_to_execute {
				break
			}
		}
		batch := gorocksdb.NewWriteBatch()
		state_db.BatchBegin(batch)
		requests := make([]state.StateTransitionParams, len(block_buf))
		for i, b := range block_buf {
			requests[i] = state.StateTransitionParams{
				Block:        (*vm.Block)(unsafe.Pointer(&b.VmBlock)),
				Uncles:       *(*[]ethash.BlockNumAndCoinbase)(unsafe.Pointer(&b.UncleBlocks)),
				Transactions: *(*[]vm.Transaction)(unsafe.Pointer(&b.Transactions)),
			}
		}
		last_block := block_buf[len(block_buf)-1]
		fmt.Println("blocks:", block_buf[0].Number, "-", last_block.Number, "tx_count:", tx_count)
		now := time.Now()
		result := state_transition_service.TransitionState(tx_count, requests...)
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
		assert.EQ(result.StateRoot.Hex(), last_block.StateRoot.Hex())
		//break
		state_db.BatchDone()
		batch.Put(bin.BytesView("last_block"), bin.ENC_b_endian_64(last_block.Number))
		util.PanicIfNotNil(db.Write(opts_w_default, batch))
		if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
			write_reset_profiles(time.Now())
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
