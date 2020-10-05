package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"time"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"

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
	const profiling = true
	const desired_num_trx_per_block = 10000
	//const desired_num_trx_per_block = 0

	usr_dir, e1 := os.UserHomeDir()
	util.PanicIfNotNil(e1)
	dest_data_dir := mkdir_all(usr_dir, "taraxa_evm_test")

	profile_basedir := mkdir_all(dest_data_dir, "profiles")
	new_prof_file := func(time time.Time, kind string) *os.File {
		ret, err := os.Create(path.Join(profile_basedir, strconv.FormatInt(time.Unix(), 10)+"_"+kind+".prof"))
		util.PanicIfNotNil(err)
		return ret
	}
	last_profile_snapshot_time := time.Now()
	if profiling {
		debug.SetGCPercent(-1)
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
	defer blk_db.Close()

	type Transaction struct {
		From     common.Address  `json:"from" gencodec:"required"`
		GasPrice *hexutil.Big    `json:"gasPrice" gencodec:"required"`
		To       *common.Address `json:"to,omitempty"`
		Nonce    hexutil.Uint64  `json:"nonce" gencodec:"required"`
		Value    *hexutil.Big    `json:"value" gencodec:"required"`
		Gas      hexutil.Uint64  `json:"gas" gencodec:"required"`
		Input    hexutil.Bytes   `json:"input" gencodec:"required"`
	}
	type Log struct {
		Address common.Address `json:"address"  gencodec:"required"`
		Topics  []common.Hash  `json:"topics"  gencodec:"required"`
		Data    hexutil.Bytes  `json:"data"  gencodec:"required"`
	}
	type Receipt struct {
		ContractAddress *common.Address `json:"contractAddress"`
		GasUsed         hexutil.Uint64  `json:"gasUsed"  gencodec:"required"`
		Logs            []Log           `json:"logs"  gencodec:"required"`
	}
	type TransactionAndReceipt struct {
		Transaction
		Receipt Receipt `json:"receipt"  gencodec:"required"`
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
		UncleBlocks  []UncleBlock            `json:"uncleBlocks"  gencodec:"required"`
		Transactions []TransactionAndReceipt `json:"transactions"  gencodec:"required"`
		Hash         common.Hash             `json:"hash" gencodec:"required"`
		StateRoot    common.Hash             `json:"stateRoot" gencodec:"required"`
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

	statedb := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: mkdir_all(dest_data_dir, "state_db"),
	})
	defer statedb.Close()

	latest_state := statedb.GetLatestState()
	SUT := new(state_transition.StateTransition).Init(
		latest_state,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(getBlockByNumber(num).Hash[:])
		},
		nil,
		state_common.ExecutionConfig{
			ETHForks: *params.MainnetChainConfig,
		},
		core.MainnetGenesisBalances(),
		state_transition.Opts{
			EVMState: state_evm.Opts{
				NumTransactionsToBuffer: desired_num_trx_per_block + 1,
			},
			Trie: state_transition.TrieSinkOpts{
				MainTrie: trie.WriterOpts{
					FullNodeLevelsToCache: 4,
				},
			},
		},
	)
	defer SUT.Close()

	last_committed_state_desc := latest_state.GetCommittedDescriptor()
	last_blk_num := last_committed_state_desc.BlockNum
	assert.EQ(getBlockByNumber(last_blk_num).StateRoot.Hex(), last_committed_state_desc.StateRoot.Hex())
	tps_sum, tps_cnt, tps_min, tps_max := 0.0, 0, math.MaxFloat64, -1.0
	for {
		blk_num_since := last_blk_num + 1
		var block_buf []*BlockInfo
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
		fmt.Println("blocks:", blk_num_since, "-", last_blk_num, "tx_count:", tx_count)
		time_before_execution := time.Now()
		for _, b := range block_buf {
			SUT.BeginBlock((*vm.BlockInfo)(unsafe.Pointer(&b.VmBlock)))
			for _, trx_and_receipt := range b.Transactions {
				res := SUT.ExecuteTransaction((*vm.Transaction)(unsafe.Pointer(&trx_and_receipt.Transaction)))
				receipt := trx_and_receipt.Receipt
				assert.EQ(uint64(receipt.GasUsed), res.GasUsed)
				assert.EQ(len(receipt.Logs), len(res.Logs))
				for i, log := range receipt.Logs {
					actual_log := res.Logs[i]
					assert.EQ(log.Address, actual_log.Address)
					assert.Holds(bytes.Equal(log.Data, actual_log.Data))
					assert.EQ(len(log.Topics), len(actual_log.Topics))
					for i, topic := range log.Topics {
						assert.EQ(topic, actual_log.Topics[i])
					}
				}
				if receipt.ContractAddress == nil {
					assert.EQ(common.ZeroAddress, res.NewContractAddr)
				} else {
					assert.EQ(*receipt.ContractAddress, res.NewContractAddr)
				}
			}
			SUT.EndBlock(*(*[]ethash.BlockNumAndCoinbase)(unsafe.Pointer(&b.UncleBlocks)))
		}
		state_root := SUT.PrepareCommit()
		assert.EQ(block_buf[len(block_buf)-1].StateRoot.Hex(), state_root.Hex())
		//return
		SUT.Commit()
		tps := float64(tx_count) / time.Now().Sub(time_before_execution).Seconds()
		tps_sum += tps
		tps_cnt++
		if tps < tps_min {
			tps_min = tps
		}
		if tps_max < tps {
			tps_max = tps
		}
		fmt.Println("TPS current:", tps, "avg:", tps_sum/float64(tps_cnt), "min:", tps_min, "max:", tps_max)
		if profiling {
			pprof.StopCPUProfile()
			fmt.Println("gc...")
			runtime.GC()
			util.PanicIfNotNil(pprof.WriteHeapProfile(new_prof_file(last_profile_snapshot_time, "heap")))
			last_profile_snapshot_time = time.Now()
			util.PanicIfNotNil(pprof.StartCPUProfile(new_prof_file(last_profile_snapshot_time, "cpu")))
		} else {
			fmt.Println("gc...")
			runtime.GC()
		}
		debug.FreeOSMemory()
	}
}

func mkdir_all(path_segments ...string) string {
	path := path.Join(path_segments...)
	util.PanicIfNotNil(os.MkdirAll(path, os.ModePerm))
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
