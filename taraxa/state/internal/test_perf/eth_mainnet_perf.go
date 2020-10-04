package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

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
	const profiling = false
	const disable_gc = profiling || false
	const desired_num_trx_per_block = 40000

	usr_dir, e1 := os.UserHomeDir()
	util.PanicIfNotNil(e1)
	dest_data_dir := mkdirp(usr_dir + "/taraxa_evm_test")

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

	blk_db_opts := gorocksdb.NewDefaultOptions()
	blk_db_opts.SetErrorIfExists(false)
	blk_db_opts.SetCreateIfMissing(true)
	blk_db_opts.SetCreateIfMissingColumnFamilies(true)
	blk_db_opts.IncreaseParallelism(runtime.NumCPU())
	blk_db_opts.SetMaxFileOpeningThreads(runtime.NumCPU())
	blk_db_opts.OptimizeForPointLookup(256)
	blk_db_opts.SetMaxOpenFiles(32)
	blk_db, e0 := gorocksdb.OpenDbForReadOnly(blk_db_opts, "/home/oleg/win10/ubuntu/blockchain", false)
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

	rocksdb_opts_r_default := gorocksdb.NewDefaultReadOptions()
	getBlockByNumber := func(block_num types.BlockNum) *BlockInfo {
		block_json, err := blk_db.GetPinned(rocksdb_opts_r_default, bin.BytesView(fmt.Sprintf("%09d", block_num)))
		util.PanicIfNotNil(err)
		ret := new(BlockInfo)
		util.PanicIfNotNil(json.Unmarshal(block_json.Data(), ret))
		block_json.Destroy()
		return ret
	}

	last_blk_num_file := path.Join(dest_data_dir, "last_blk")
	last_blk_num := read_last_block_n(last_blk_num_file)
	is_genesis := last_blk_num == types.BlockNumberNIL
	var last_root common.Hash
	if is_genesis {
		last_blk_num = 0
	} else {
		last_root = getBlockByNumber(last_blk_num).StateRoot
	}

	SUT := new(state_transition.StateTransition).Init(
		new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
			Path: mkdirp(dest_data_dir + "/state_db"),
		}),
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(getBlockByNumber(num).Hash[:])
		},
		state_common.ChainConfig{
			ExecutionConfig: state_common.ExecutionConfig{
				ETHForks: *params.MainnetChainConfig,
			},
		},
		last_blk_num,
		&last_root,
		state_transition.Opts{
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
		root := SUT.apply_genesis(state_transition.GenesisConfig{Balances: core.MainnetGenesisBalances()})
		assert.EQ(root.Hex(), getBlockByNumber(0).StateRoot.Hex())
		write_last_block_n(last_blk_num_file, 0)
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
		fmt.Println("blocks:", block_num_from, "-", last_blk_num, "tx_count:", tx_count)
		now := time.Now()
		for _, b := range block_buf {
			SUT.BeginBlock((*vm.BlockInfo)(unsafe.Pointer(&b.VmBlock)))
			for i := range b.Transactions {
				SUT.ExecuteTransaction((*vm.Transaction)(unsafe.Pointer(&b.Transactions[i])))
			}
			SUT.EndBlock(*(*[]ethash.BlockNumAndCoinbase)(unsafe.Pointer(&b.UncleBlocks)))
		}
		assert.EQ(SUT.Commit().Hex(), block_buf[len(block_buf)-1].StateRoot.Hex())
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
		write_last_block_n(last_blk_num_file, last_blk_num)

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

func write_file(fname string, val []byte) {
	util.PanicIfNotNil(ioutil.WriteFile(fname, val, 0644))
}

func read_file(fname string) []byte {
	ret, err := ioutil.ReadFile(fname)
	if os.IsNotExist(err) {
		return nil
	}
	util.PanicIfNotNil(err)
	return ret
}

func read_last_block_n(fname string) types.BlockNum {
	bytes := read_file(fname)
	if len(bytes) == 0 {
		return types.BlockNumberNIL
	}
	return bin.DEC_b_endian_64(bytes)
}

func write_last_block_n(fname string, n types.BlockNum) {
	write_file(fname, bin.ENC_b_endian_64(n))
}
