package state

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/data"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_concurrent_schedule"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/tecbot/gorocksdb"
	"math/big"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"
)

func TestEthMainnetSmoke(t *testing.T) {
	dest_data_dir := path.Join(os.TempDir(), strconv.Itoa(rand.Int()))
	util.PanicIfNotNil(os.RemoveAll(dest_data_dir))
	mkdirp(dest_data_dir)
	defer os.RemoveAll(dest_data_dir)

	genesis_accounts := data.Parse_eth_mainnet_genesis_accounts()
	blocks := data.Parse_eth_mainnet_blocks_0_300000()

	opts_w_default := gorocksdb.NewDefaultWriteOptions()
	statedb_opts := gorocksdb.NewDefaultOptions()
	statedb_opts.SetErrorIfExists(false)
	statedb_opts.SetCreateIfMissing(true)
	statedb_opts.SetCreateIfMissingColumnFamilies(true)
	const col_cnt = 1 + state_db_rocksdb.COL_COUNT
	cfnames, cfopts := [col_cnt]string{"default"}, [col_cnt]*gorocksdb.Options{gorocksdb.NewDefaultOptions()}
	for i := state_db_rocksdb.Column(1); i < col_cnt; i++ {
		cfnames[i], cfopts[i] = strconv.Itoa(int(i)), gorocksdb.NewDefaultOptions()
	}
	statedb_rocksdb, cols, err_1 := gorocksdb.OpenDbColumnFamilies(statedb_opts, dest_data_dir, cfnames[:], cfopts[:])
	util.PanicIfNotNil(err_1)
	defer statedb_rocksdb.Close()
	var state_db_cols state_db_rocksdb.Columns
	copy(state_db_cols[:], cols[1:])
	state_db := new(state_db_rocksdb.DB).Init(statedb_rocksdb, state_db_cols)

	SUT := new(state_transition.StateTransition).Init(
		state_db,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(blocks[num].Hash[:])
		},
		state_common.ChainConfig{
			EVMChainConfig: state_common.EVMChainConfig{
				ETHChainConfig: *params.MainnetChainConfig,
			},
		},
		0,
		common.Hash{},
		state_transition.CacheOpts{
			MainTrieWriterOpts: trie.WriterCacheOpts{
				FullNodeLevelsToCache: 5,
				ExpectedDepth:         trie.MaxDepth,
			},
			AccTrieWriterOpts: trie.WriterCacheOpts{
				ExpectedDepth: 16,
			},
			ExpectedMaxNumTrxPerBlock: 100,
		},
	)
	batch := gorocksdb.NewWriteBatch()
	state_db.TransactionBegin(batch)
	root := SUT.ApplyAccounts(genesis_accounts)
	assert.EQ(root.Hex(), blocks[0].StateRoot.Hex())
	state_db.TransactionEnd()
	util.PanicIfNotNil(statedb_rocksdb.Write(opts_w_default, batch))
	state_db.Refresh()

	blk_num := 1
	defer fmt.Println("# executed blocks:", blk_num)
	for ; blk_num < len(blocks); blk_num++ {
		blk := blocks[blk_num]
		batch := gorocksdb.NewWriteBatch()
		state_db.TransactionBegin(batch)
		result := SUT.ApplyBlock(
			&blk.EVMBlock,
			blk.Transactions,
			blk.UncleBlocks,
			state_concurrent_schedule.ConcurrentSchedule{},
		)
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
