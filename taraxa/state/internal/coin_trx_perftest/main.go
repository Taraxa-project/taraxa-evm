package main

import (
	"flag"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"time"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/jsonutil"

	"github.com/tecbot/gorocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/Taraxa-project/taraxa-evm/core/vm"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/files"
)

// Usage: `go run <this_file> ...arguments`. Also can be just compiled as usual via `go build` and run as executable.
// Use `--help` argument to get the help.
// TODO more workload profiles
func main() {
	flag.Usage = func() {
		flag.PrintDefaults()
		fmt.Println("Stats are also written line-per-record in json file: <output_dir>/<tag>_stats.json")
	}
	var output_dir string
	flag.StringVar(&output_dir, "output_dir", path.Join(os.TempDir(), "taraxa_evm_perftest"), ""+
		"Base directory for all output")
	var num_addrs uint64
	flag.Uint64Var(&num_addrs, "num_addrs", 10e6, ""+
		"Preparation option. How many addresses to generate. Must be divisible by <num_prepare_blocks>.")
	var num_prepare_blocks types.BlockNum
	flag.Uint64Var(&num_prepare_blocks, "num_prepare_blocks", 1e6, ""+
		"Preparation option. over how many blocks <num_addrs> addresses will be generated")
	var prepare_trx_batch_size uint64
	flag.Uint64Var(&prepare_trx_batch_size, "prepare_trx_batch_size", 10e3, ""+
		"Preparation option. How many transactions to batch before execution in the preparation phase."+
		"If the batch is too small, it will go slower.")
	var tag string
	flag.StringVar(&tag, "tag", "default_tag", ""+
		"A classifier for all post-preparation (test phase) output files. Can be used to create a common base data "+
		"via preparation, and then run several modes of testing with different options")
	var num_test_blocks types.BlockNum
	flag.Uint64Var(&num_test_blocks, "num_test_blocks", 1e3, ""+
		"How many test blocks to execute")
	var test_trx_batch_size uint64
	flag.Uint64Var(&test_trx_batch_size, "test_trx_batch_size", 20e3, ""+
		"How many transactions transactions to put in a test block")
	var disable_gc bool
	flag.BoolVar(&disable_gc, "disable_gc", true, ""+
		"Whether to disable garbage collection during a testing run and explicitly call GC in-between runs")
	var enable_profiling bool
	flag.BoolVar(&enable_profiling, "enable_profiling", false, ""+
		"Whether to perform profiling. Implies <disable_gc>. "+
		"Two types of profiles (cpu, heap) will be written per test block. "+
		"Profiling data can be found at <output_dir>/<tag>_prof numbered by block numbers. (default false)")
	var purge_stats bool
	flag.BoolVar(&purge_stats, "purge_stats", false, ""+
		"Whether to clean the stats collected so far. This will trigger re-running "+
		"test phase from the beginning but keep the test state db as is. (default false)")
	var purge_testdb bool
	flag.BoolVar(&purge_testdb, "purge_testdb", false, ""+
		"Whether to clean the test state db. Implies <purge_stats>. (default false)")

	flag.Parse()
	asserts.Holds(uint64(num_addrs)%num_prepare_blocks == 0)
	purge_stats = purge_testdb || purge_stats
	disable_gc = enable_profiling || disable_gc

	profile_basedir := files.CreateDirectories(output_dir, tag+"_prof")
	make_prof_file := func(i uint64, kind string) *os.File {
		ret, err := os.Create(path.Join(profile_basedir, fmt.Sprint(i)+"_"+kind+".prof"))
		util.PanicIfNotNil(err)
		return ret
	}

	rocksdb_opts_r_default := gorocksdb.NewDefaultReadOptions()
	rocksdb_opts_w_default := gorocksdb.NewDefaultWriteOptions()
	statedb_prep_path := files.Path(output_dir, "statedb_prep")
	statedb_prep := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: statedb_prep_path,
	})
	root_genesis_bal := new(big.Int).Set(bigutil.MaxU256)
	OpenStateAPI := func(db state_db.DB) *state.API {
		return new(state.API).Init(
			db,
			func(num types.BlockNum) *big.Int { panic("unexpected") },
			state.ChainConfig{
				DisableBlockRewards: true,
				ExecutionOptions: vm.ExecutionOpts{
					DisableGasFee:     false,
					DisableNonceCheck: true,
				},
				ETHChainConfig: params.ChainConfig{
					DAOForkBlock: types.BlockNumberNIL,
				},
				GenesisBalances: core.BalanceMap{
					tests.Addr(1): root_genesis_bal,
				},
			},
			state.APIOpts{
				ExpectedMaxTrxPerBlock:        test_trx_batch_size,
				MainTrieFullNodeLevelsToCache: 5,
			},
		)
	}
	state_api := OpenStateAPI(statedb_prep)
	state_desc := state_api.GetCommittedStateDescriptor()
	st := state_api.GetStateTransition()

	const trx_gas = 210000
	addr_per_blk := uint64(num_addrs) / num_prepare_blocks
	evm_blk_info := &vm.BlockInfo{}
	target_bal_per_addr := new(big.Int).Div(root_genesis_bal, new(big.Int).SetUint64(uint64(num_addrs)))
	for blk_n, batch_size := state_desc.BlockNum, uint64(0); blk_n < num_prepare_blocks; blk_n++ {
		fmt.Println("preparation block #", blk_n, "of", num_prepare_blocks)
		st.BeginBlock(evm_blk_info)
		for j := uint64(0); j < addr_per_blk; j++ {
			st.ExecuteTransaction(&vm.Transaction{
				From:     tests.Addr(1),
				To:       tests.AddrP(2 + blk_n*addr_per_blk + j),
				Value:    target_bal_per_addr,
				GasPrice: bigutil.Big0,
				Gas:      trx_gas,
			})
			batch_size++
		}
		st.EndBlock(nil)
		if batch_size >= prepare_trx_batch_size || blk_n == num_prepare_blocks-1 {
			st.Commit()
			batch_size = 0
		}
	}
	state_api.Close()
	statedb_prep.Close()

	statedb_path := files.Path(output_dir, tag+"_statedb")
	if purge_testdb {
		files.RemoveAll(statedb_path)
	}
	if !files.Exists(statedb_path) {
		fmt.Println("copying the prep db to use it in the testing phase...")
		files.Copy(statedb_prep_path, statedb_path)
		fmt.Println("done")
	}
	statedb := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: statedb_path,
	})
	defer statedb.Close()

	stats_db_path := files.Path(output_dir, tag+"_stats")
	stats_file_path := files.Path(output_dir, tag+"_stats.json")
	if purge_stats {
		files.RemoveAll(stats_file_path)
		files.RemoveAll(stats_db_path)
	}
	stats_file, err := os.OpenFile(stats_file_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	util.PanicIfNotNil(err)
	defer func() { util.PanicIfNotNil(stats_file.Close()) }()

	stats_db_opts := gorocksdb.NewDefaultOptions()
	stats_db_opts.SetErrorIfExists(false)
	stats_db_opts.SetCreateIfMissing(true)
	stats_db, err_0 := gorocksdb.OpenDb(stats_db_opts, stats_db_path)
	util.PanicIfNotNil(err_0)
	defer stats_db.Close()

	state_api = OpenStateAPI(statedb)
	state_desc = state_api.GetCommittedStateDescriptor()
	st = state_api.GetStateTransition()
	defer state_api.Close()

	var test_blk_ordinal types.BlockNum
	type StatsRecord struct{ Curr, Cumulative, Min, Max, Mean float64 }
	stats := make(map[string]*StatsRecord)
	util.Call(func() {
		itr := stats_db.NewIterator(rocksdb_opts_r_default)
		defer itr.Close()
		if itr.SeekToLast(); itr.Valid() {
			test_blk_ordinal = 1 + bin.DEC_b_endian_64(itr.Key().Data())
			itr.Key().Free()
			jsonutil.MustDecode(itr.Value().Data(), &stats)
			itr.Value().Free()
		}
		util.PanicIfNotNil(itr.Err())
	})
	elapsed_per_stage := make(map[string]float64)
	with_timer := func(name string, action func()) {
		time_start := time.Now()
		action()
		elapsed_per_stage[name] = time.Now().Sub(time_start).Seconds()
	}
	if disable_gc {
		debug.SetGCPercent(-1)
	}
	for ; test_blk_ordinal < num_test_blocks; test_blk_ordinal++ {
		fmt.Println("running block #", test_blk_ordinal, "of", num_test_blocks, "...")

		if enable_profiling {
			pprof.StartCPUProfile(make_prof_file(test_blk_ordinal, "cpu"))
		}

		with_timer("execution", func() {
			st.BeginBlock(evm_blk_info)
			for i := uint64(0); i < test_trx_batch_size; i++ {
				from := tests.Addr(uint64(2 + rand.Int63n(int64(num_addrs))))
				to := tests.Addr(uint64(2 + rand.Int63n(int64(num_addrs))))
				trx := vm.Transaction{
					From:     from,
					To:       &to,
					Value:    bigutil.Big1,
					GasPrice: bigutil.Big0,
					Gas:      trx_gas,
				}
				st.ExecuteTransaction(&trx)
			}
			st.EndBlock(nil)
		})
		with_timer("trie_commit", func() {
			st.PrepareCommit()
		})
		with_timer("db_commit", func() {
			st.Commit()
		})

		if enable_profiling {
			pprof.StopCPUProfile()
		}
		if disable_gc {
			debug.FreeOSMemory()
		}
		if enable_profiling {
			util.PanicIfNotNil(pprof.WriteHeapProfile(make_prof_file(test_blk_ordinal, "heap")))
			util.PanicIfNotNil(pprof.StartCPUProfile(make_prof_file(test_blk_ordinal, "cpu")))
		}

		var elapsed_total float64
		for _, elapsed := range elapsed_per_stage {
			elapsed_total += elapsed
		}
		data_points := make(map[string]float64, len(elapsed_per_stage)+2)
		data_points["execution (TPS)"] = float64(test_trx_batch_size) / elapsed_per_stage["execution"]
		data_points["total (TPS)"] = float64(test_trx_batch_size) / elapsed_total
		for name, val := range elapsed_per_stage {
			data_points[name+" %"] = 100 * val / elapsed_total
			delete(elapsed_per_stage, name)
		}
		for name, val := range data_points {
			stage_stats := stats[name]
			if stage_stats == nil {
				stage_stats = &StatsRecord{Max: -1, Min: math.MaxFloat64}
				stats[name] = stage_stats
			}
			stage_stats.Curr = val
			stage_stats.Cumulative += val
			stage_stats.Min = math.Min(stage_stats.Min, val)
			stage_stats.Max = math.Max(stage_stats.Max, val)
			stage_stats.Mean = stage_stats.Cumulative / float64(test_blk_ordinal+1)
			delete(data_points, name)
		}
		stats_json := jsonutil.MustEncodePretty(stats, "    ")
		stats_db.Put(rocksdb_opts_w_default, bin.ENC_b_endian_64(test_blk_ordinal), stats_json)
		fmt.Println("stats:", bin.StringView(stats_json))
		_, err := stats_file.Write(append(jsonutil.MustEncode(stats), '\n'))
		util.PanicIfNotNil(err)
	}
}
