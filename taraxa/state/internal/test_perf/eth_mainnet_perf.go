package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_config"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
)

func main() {
	var last_block_key = bin.BytesView("last_block")

	const profiling = false
	const disable_gc = profiling || false

	d, e1 := os.UserHomeDir()
	util.PanicIfNotNil(e1)

	const desired_num_trx_per_block = 0
	dest_data_dir := mkdirp(d + "/taraxa_evm_test")

	if disable_gc {
		debug.SetGCPercent(-1)
	}
	var max_heap_size uint64
	var mem_stats runtime.MemStats
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
	blk_db_opts.OptimizeForPointLookup(256)
	blk_db_opts.SetMaxOpenFiles(32)
	blk_db, e0 := gorocksdb.OpenDbForReadOnly(
		blk_db_opts,
		"/home/oleg/win10/ubuntu/blockchain",
		false)
	util.PanicIfNotNil(e0)

	type Transaction struct {
		From     common.Address  `json:"from" gencodec:"required"`
		GasPrice *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		To       *common.Address `json:"to,omitempty"`
		Nonce    hexutil.Uint64  `json:"nonce" gencodec:"required"`
		Value    *hexutil.Big    `json:"value" gencodec:"required"`
		Gas      hexutil.Uint64  `json:"gas" gencodec:"required"`
		Input    hexutil.Bytes   `json:"input" gencodec:"required"`
	}
	type UncleBlock struct {
		Number hexutil.Uint64 `json:"number"  gencodec:"required"`
		Miner  common.Address `json:"miner"  gencodec:"required"`
	}
	type VmBlock struct {
		Miner      common.Address `json:"miner" gencodec:"required"`
		GasLimit   hexutil.Uint64 `json:"gasLimit"  gencodec:"required"`
		Time       hexutil.Uint64 `json:"timestamp"  gencodec:"required"`
		Difficulty *hexutil.Big   `json:"difficulty"  gencodec:"required"`
	}
	type BlockInfo struct {
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
	const col_cnt = 1 + state_db_rocksdb.COL_COUNT
	cfnames, cfopts := [col_cnt]string{"default"}, [col_cnt]*gorocksdb.Options{gorocksdb.NewDefaultOptions()}
	for i := state_db_rocksdb.Column(1); i < col_cnt; i++ {
		opts := gorocksdb.NewDefaultOptions()
		switch i - 1 {
		case state_db_rocksdb.COL_main_trie_value, state_db_rocksdb.COL_acc_trie_value:
		default:
			opts.OptimizeForPointLookup(512)
		}
		cfnames[i], cfopts[i] = strconv.Itoa(int(i)), opts
	}
	statedb_rocksdb, cols, err_1 := gorocksdb.OpenDbColumnFamilies(
		statedb_opts,
		mkdirp(dest_data_dir+"/state_db"),
		cfnames[:],
		cfopts[:])
	util.PanicIfNotNil(err_1)
	defer statedb_rocksdb.Close()
	var state_db state_db_rocksdb.DB
	var state_db_cols state_db_rocksdb.Columns
	copy(state_db_cols[:], cols[1:])
	state_db.Init(statedb_rocksdb, state_db_cols)

	var last_blk_num types.BlockNum
	var last_root common.Hash
	last_block_num_b, err := statedb_rocksdb.GetBytes(opts_r_default, last_block_key)
	util.PanicIfNotNil(err)
	is_genesis := len(last_block_num_b) == 0
	if !is_genesis {
		last_blk_num = bin.DEC_b_endian_64(last_block_num_b)
		last_root = getBlockByNumber(last_blk_num).StateRoot
	}

	SUT := new(state_transition.StateTransition).Init(
		&state_db,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(getBlockByNumber(num).Hash[:])
		},
		state_config.ChainConfig{
			Execution: state_config.ExecutionConfig{
				ETHForks: *params.MainnetChainConfig,
			},
		},
		last_blk_num,
		&last_root,
		state_transition.StateTransitionOpts{
			TrieWriters: state_transition.TrieWriterOpts{
				MainTrieWriterOpts: trie.WriterCacheOpts{
					FullNodeLevelsToCache: 16,
					ExpectedDepth:         trie.MaxDepth,
				},
				AccTrieWriterOpts: trie.WriterCacheOpts{
					ExpectedDepth: 20,
				},
			},
			ExpectedMaxNumTrxPerBlock: 80000,
		},
	)

	if is_genesis {
		batch := gorocksdb.NewWriteBatch()
		state_db.BatchBegin(batch)
		root := SUT.GenesisInit(state_transition.GenesisConfig{Balances: core.MainnetGenesisBalances()})
		assert.EQ(root.Hex(), getBlockByNumber(0).StateRoot.Hex())
		batch.Put(last_block_key, bin.ENC_b_endian_64(0))
		state_db.BatchEnd()
		util.PanicIfNotNil(statedb_rocksdb.Write(opts_w_default, batch))
		state_db.Refresh()
	}

	tps_sum, tps_cnt, tps_min, tps_max := 0.0, 0, math.MaxFloat64, -1.0
	var block_buf []*BlockInfo
	for {
		block_buf = block_buf[:0]
		block_num_from := last_blk_num + 1
		tx_count := 0
		for {
			last_blk_num++
			last_block := getBlockByNumber(last_blk_num)
			block_buf = append(block_buf, last_block)
			tx_count += len(last_block.Transactions)
			if tx_count >= desired_num_trx_per_block {
				break
			}
		}
		batch := gorocksdb.NewWriteBatch()
		state_db.BatchBegin(batch)
		fmt.Println("blocks:", block_num_from, "-", last_blk_num, "tx_count:", tx_count)
		now := time.Now()
		for _, b := range block_buf {
			SUT.BeginBlock((*vm.BlockWithoutNumber)(unsafe.Pointer(&b.VmBlock)))
			for i := range b.Transactions {
				SUT.SubmitTransaction((*vm.Transaction)(unsafe.Pointer(&b.Transactions[i])))
			}
			SUT.EndBlock(*(*[]ethash.BlockNumAndCoinbase)(unsafe.Pointer(&b.UncleBlocks)))
		}
		result := SUT.CommitSync()
		assert.EQ(result.StateRoot.Hex(), block_buf[len(block_buf)-1].StateRoot.Hex())
		//return
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
		state_db.BatchEnd()
		batch.Put(last_block_key, bin.ENC_b_endian_64(last_blk_num))
		util.PanicIfNotNil(statedb_rocksdb.Write(opts_w_default, batch))
		batch.Destroy()
		state_db.Refresh()

		if disable_gc {
			if runtime.ReadMemStats(&mem_stats); mem_stats.HeapAlloc > max_heap_size {
				fmt.Println("gc...")
				runtime.GC()
				if profiling {
					pprof.StopCPUProfile()
					util.PanicIfNotNil(pprof.WriteHeapProfile(new_prof_file(last_profile_snapshot_time, "heap")))
					last_profile_snapshot_time = time.Now()
					util.PanicIfNotNil(pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu")))
				}
				runtime.ReadMemStats(&mem_stats)
				max_heap_size = mem_stats.HeapAlloc * 4
			}
		}
	}
}

func mkdirp(path string) string {
	util.PanicIfNotNil(exec.Command("mkdir", "-p", path).Run())
	return path
}
