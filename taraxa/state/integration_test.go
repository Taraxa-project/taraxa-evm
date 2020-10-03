package state

import (
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"

	"github.com/schollz/progressbar/v3"

	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/data"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
)

func TestEthMainnetSmoke(t *testing.T) {
	dest_data_dir := path.Join(os.TempDir(), strconv.Itoa(rand.Int()))
	util.PanicIfNotNil(os.RemoveAll(dest_data_dir))
	mkdirp(dest_data_dir)
	defer os.RemoveAll(dest_data_dir)
	blocks := data.Parse_eth_mainnet_blocks_0_300000()
	SUT := new(state_transition.StateTransition).Init(
		new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
			Path: dest_data_dir,
		}),
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(blocks[num].Hash[:])
		},
		state_common.ChainConfig{
			Execution: state_common.ExecutionConfig{
				ETHForks: *params.MainnetChainConfig,
			},
		},
		0,
		nil,
		state_transition.StateTransitionOpts{
			TrieWriters: state_transition.TrieWriterOpts{
				MainTrieWriterOpts: trie.WriterCacheOpts{
					FullNodeLevelsToCache: 5,
					ExpectedDepth:         trie.MaxDepth,
				},
				AccTrieWriterOpts: trie.WriterCacheOpts{
					ExpectedDepth: 16,
				},
			},
			ExpectedMaxNumTrxPerBlock: 500,
		},
	)
	root := SUT.GenesisInit(state_transition.GenesisConfig{Balances: core.MainnetGenesisBalances()})
	assert.EQ(root.Hex(), blocks[0].StateRoot.Hex())
	progress_bar := progressbar.Default(int64(len(blocks)))
	defer progress_bar.Finish()
	for blk_num := 1; blk_num < len(blocks); blk_num++ {
		blk := blocks[blk_num]
		SUT.BeginBlock(&blk.EVMBlock)
		for i := range blk.Transactions {
			SUT.ExecuteTransaction(&blk.Transactions[i])
		}
		SUT.EndBlock(blk.UncleBlocks)
		assert.EQ(SUT.Commit().Hex(), blk.StateRoot.Hex())
		progress_bar.Add(1)
	}
}

func mkdirp(path string) string {
	util.PanicIfNotNil(exec.Command("mkdir", "-p", path).Run())
	return path
}
