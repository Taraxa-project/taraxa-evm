package state

import (
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"

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
	statedb := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: dest_data_dir,
	})
	defer statedb.Close()
	SUT := new(state_transition.StateTransition).Init(
		statedb.GetLatestState(),
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(blocks[num].Hash[:])
		},
		nil,
		state_common.ExecutionConfig{
			ETHForks: *params.MainnetChainConfig,
		},
		core.MainnetGenesisBalances(),
		state_transition.Opts{
			EVMState: state_evm.Opts{
				NumTransactionsToBuffer: 300,
			},
			Trie: state_transition.TrieSinkOpts{
				MainTrie: trie.WriterOpts{
					FullNodeLevelsToCache: 4,
				},
			},
		},
	)
	assert.EQ(statedb.GetLatestState().GetCommittedDescriptor().StateRoot.Hex(), blocks[0].StateRoot.Hex())
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
