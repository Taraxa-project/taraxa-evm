//+build none

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
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
	"math/big"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"unsafe"
)

// Example: go run eth_mainnet.go "{\"BlockDBPath\": \"/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain/\", \"NumBlocksToExecute\": 2000000 }"
func main() {
	var param struct {
		BlockDBPath        string         `gencodec:"required"`
		NumBlocksToExecute types.BlockNum `gencodec:"required"`
		DestDataDir        string
	}
	util.PanicIfNotNil(json.Unmarshal(bin.BytesView(os.Args[1]), &param))
	if len(param.DestDataDir) == 0 {
		param.DestDataDir = os.TempDir() + strconv.Itoa(rand.Int())
		fmt.Println("using random output dir:", param.DestDataDir)
	}
	util.PanicIfNotNil(os.RemoveAll(param.DestDataDir))

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
	blk_db, err_0 := gorocksdb.OpenDbForReadOnly(blk_db_opts, param.BlockDBPath, false)
	util.PanicIfNotNil(err_0)
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
		Number     types.BlockNum `json:"number" gencodec:"required"`
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
	block_chan := make(chan *BlockInfo, 5)
	block_load_requests := make(chan byte, cap(block_chan))
	defer close(block_load_requests)
	go func() {
		defer close(block_chan)
		defer blk_db.Close()
		next_to_load := types.BlockNum(1)
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
		mkdirp(param.DestDataDir),
		cfnames[:],
		cfopts[:])
	util.PanicIfNotNil(err_1)
	defer statedb_rocksdb.Close()
	var state_db state_db_rocksdb.DB
	var state_db_cols state_db_rocksdb.Columns
	copy(state_db_cols[:], cols[1:])
	state_db.Init(statedb_rocksdb, state_db_cols)

	var SUT state_transition.StateTransition
	SUT.Init(
		&state_db,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(getBlockByNumber(num).Hash[:])
		},
		common.Hash{},
		state_common.ChainConfig{
			EVMChainConfig: state_common.EVMChainConfig{
				ETHChainConfig: *params.MainnetChainConfig,
			},
		},
		state_transition.CacheOpts{
			MainTrieWriterOpts: trie.WriterCacheOpts{
				FullNodeLevelsToCache: 5,
				ExpectedDepth:         trie.MaxDepth,
			},
			AccTrieWriterOpts: trie.WriterCacheOpts{
				ExpectedDepth: 16,
			},
			ExpectedMaxNumTrxPerBlock: 400,
		},
	)
	batch := gorocksdb.NewWriteBatch()
	state_db.TransactionBegin(batch)
	root := SUT.ApplyGenesis(core.MainnetGenesis().Alloc)
	assert.EQ(root.Hex(), getBlockByNumber(0).StateRoot.Hex())
	state_db.TransactionEnd()
	util.PanicIfNotNil(statedb_rocksdb.Write(opts_w_default, batch))
	state_db.Refresh()

	for i := types.BlockNum(0); i < param.NumBlocksToExecute; i++ {
		block_load_requests <- 0
		blk := <-block_chan
		tx_count := len(blk.Transactions)
		batch := gorocksdb.NewWriteBatch()
		state_db.TransactionBegin(batch)
		fmt.Println("block", blk.Number, "tx_count:", tx_count)
		result := SUT.Apply(state_transition.Params{
			Block:        (*vm.Block)(unsafe.Pointer(&blk.VmBlock)),
			Uncles:       *(*[]ethash.BlockNumAndCoinbase)(unsafe.Pointer(&blk.UncleBlocks)),
			Transactions: *(*[]vm.Transaction)(unsafe.Pointer(&blk.Transactions)),
		})
		assert.EQ(result.StateRoot.Hex(), blk.StateRoot.Hex())
		state_db.TransactionEnd()
		util.PanicIfNotNil(statedb_rocksdb.Write(opts_w_default, batch))
		batch.Destroy()
		state_db.Refresh()
	}
}

func mkdirp(path string) string {
	util.PanicIfNotNil(exec.Command("mkdir", "-p", path).Run())
	return path
}
