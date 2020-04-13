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
	"github.com/Taraxa-project/taraxa-evm/dbg"
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
	var last_block_key = bin.BytesView("last_block")

	var profiling = false
	var disable_gc = true

	//usr_dir, err := os.UserHomeDir()
	//util.PanicIfNotNil(err)
	dest_data_dir := mkdirp("/Users/compuktor/taraxa_evm_data")

	if disable_gc {
		debug.SetGCPercent(-1)
	}
	//var max_heap_size uint64
	//var mem_stats runtime.MemStats

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

	opts_w_default := gorocksdb.NewDefaultWriteOptions()
	opts_r_default := gorocksdb.NewDefaultReadOptions()
	blk_db_opts := gorocksdb.NewDefaultOptions()
	blk_db_opts.SetErrorIfExists(false)
	blk_db_opts.SetCreateIfMissing(true)
	blk_db_opts.SetCreateIfMissingColumnFamilies(true)
	blk_db_opts.IncreaseParallelism(runtime.NumCPU())
	blk_db_opts.SetMaxFileOpeningThreads(runtime.NumCPU())
	//blk_db_opts.SetMaxOpenFiles(8192)
	//blk_db_opts.OptimizeForPointLookup(1024)
	blk_db_opts.SetMaxOpenFiles(1024)
	blk_db, e0 := gorocksdb.OpenDbForReadOnly(
		blk_db_opts,
		"/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain",
		false)
	util.PanicIfNotNil(e0)

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
		block_json, err := blk_db.GetPinned(opts_r_default, bin.BytesView(fmt.Sprintf("%09d", block_num)))
		util.PanicIfNotNil(err)
		ret := new(BlockInfo)
		util.PanicIfNotNil(json.Unmarshal(block_json.Data(), ret))
		block_json.Destroy()
		return ret
	}

	statedb_opts := gorocksdb.NewDefaultOptions()
	statedb_opts.SetErrorIfExists(false)
	statedb_opts.SetCreateIfMissing(true)
	statedb_opts.SetCreateIfMissingColumnFamilies(true)
	statedb_opts.IncreaseParallelism(runtime.NumCPU())
	statedb_opts.SetMaxFileOpeningThreads(runtime.NumCPU())
	statedb_opts.SetMaxOpenFiles(8192)
	const col_cnt = 8
	cfnames, cfopts := [col_cnt]string{}, [col_cnt]*gorocksdb.Options{}
	for i := 0; i < col_cnt; i++ {
		if i == 0 {
			cfnames[i] = "default"
		} else {
			cfnames[i] = strconv.Itoa(i)
		}
		cfopts[i] = gorocksdb.NewDefaultOptions()
	}
	db, cols, e1 := gorocksdb.OpenDbColumnFamilies(statedb_opts, mkdirp(dest_data_dir+"/WIP_1"), cfnames[:], cfopts[:])
	util.PanicIfNotNil(e1)
	defer db.Close()
	var state_db state_rocksdb.RocksDBStateDB
	state_db.Init(db, state_rocksdb.Columns{
		COL_code:                   cols[1],
		COL_main_trie_node:         cols[2],
		COL_main_trie_value:        cols[3],
		COL_main_trie_value_latest: cols[4],
		COL_acc_trie_node:          cols[5],
		COL_acc_trie_value:         cols[6],
		COL_acc_trie_value_latest:  cols[7],
	})

	var last_blk_num types.BlockNum
	var last_root *common.Hash
	last_block_num_b, err := db.GetBytes(opts_r_default, last_block_key)
	util.PanicIfNotNil(err)
	if len(last_block_num_b) != 0 {
		//last_blk_num = 1000000
		last_blk_num = bin.DEC_b_endian_64(last_block_num_b)
		last_root = &getBlockByNumber(last_blk_num).StateRoot
	}
	//for i := types.BlockNum(2670000); ; i++ {
	//	last_blk_num = i
	//	last_root = &getBlockByNumber(last_blk_num).StateRoot
	//	if len(state_db.GetMainTrieNode(last_root)) != 0 {
	//		break
	//	}
	//}

	var state_transition_service state.StateTransitionService
	state_transition_service.Init(
		&state_db,
		last_root,
		trie.TrieWriterOpts{
			FullNodeLevelsToCache: 5,
			AnticipatedDepth:      trie.MaxDepth,
		},
		vm.ExecutionOptions{},
		false,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(getBlockByNumber(num).Hash[:])
		}, *params.MainnetChainConfig,
	)
	if last_root == nil {
		batch := gorocksdb.NewWriteBatch()
		state_db.TransactionBegin(batch)
		root := state_transition_service.ApplyGenesis(core.MainnetGenesis().Alloc)
		assert.EQ(root.Hex(), getBlockByNumber(0).StateRoot.Hex())
		batch.Put(last_block_key, bin.ENC_b_endian_64(0))
		state_db.TransactionEnd()
		util.PanicIfNotNil(db.Write(opts_w_default, batch))
	}

	min_tx_to_execute := 20000
	block_chan := make(chan *BlockInfo, min_tx_to_execute/25+2)
	block_load_requests := make(chan byte, cap(block_chan))
	defer close(block_load_requests)
	go func() {
		defer close(block_chan)
		defer blk_db.Close()
		next_to_load := last_blk_num + 1
		for i := 0; i < cap(block_chan); i++ {
			block_chan <- getBlockByNumber(next_to_load)
			next_to_load++
		}
		for {
			if _, ok := <-block_load_requests; !ok {
				break
			}
			block_chan <- getBlockByNumber(next_to_load)
			next_to_load++
		}
	}()

	dbg.DebugState = true
	dbg.DebugEVM = true

	tps_sum, tps_cnt, tps_min, tps_max := 0.0, 0, math.MaxFloat64, -1.0
	block_buf := make([]*BlockInfo, 0, cap(block_chan))
	for {
		tx_count := 0
		for {
			block_load_requests <- 0
			last_block := <-block_chan
			block_buf = append(block_buf, last_block)
			tx_count += len(last_block.Transactions)
			if tx_count >= min_tx_to_execute {
				break
			}
		}
		batch := gorocksdb.NewWriteBatch()
		state_db.TransactionBegin(batch)
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
		state_db.TransactionEnd()
		batch.Put(last_block_key, bin.ENC_b_endian_64(last_block.Number))
		util.PanicIfNotNil(db.Write(opts_w_default, batch))
		batch.Destroy()
		state_db.Refresh()

		if disable_gc {
			fmt.Println("gc...")
			runtime.GC()
			//if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
			//
			//	runtime.GC()
			//	runtime.ReadMemStats(&mem_stats)
			//	max_heap_size = mem_stats.HeapAlloc * 4
			//}
		}
		if profiling {
			pprof.StopCPUProfile()
			util.PanicIfNotNil(pprof.WriteHeapProfile(new_prof_file(last_profile_snapshot_time, "heap")))
			last_profile_snapshot_time = time.Now()
			util.PanicIfNotNil(pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu")))
		}
	}
}

func mkdirp(path string) string {
	util.PanicIfNotNil(exec.Command("mkdir", "-p", path).Run())
	return path
}
