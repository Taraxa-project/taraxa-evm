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

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/data"
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
	api := new(API).Init(
		statedb,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(blocks[num].Hash[:])
		},
		ChainConfig{
			ETHChainConfig:  *params.MainnetChainConfig,
			GenesisBalances: core.MainnetGenesisBalances(),
		},
		Opts{
			ExpectedMaxTrxPerBlock:        300,
			MainTrieFullNodeLevelsToCache: 4,
		},
	)
	defer api.Close()
	st := api.GetStateTransition()
	assert.EQ(statedb.GetLatestState().GetCommittedDescriptor().StateRoot.Hex(), blocks[0].StateRoot.Hex())
	progress_bar := progressbar.Default(int64(len(blocks)))
	defer progress_bar.Finish()
	for blk_num := 1; blk_num < len(blocks); blk_num++ {
		blk := blocks[blk_num]
		st.BeginBlock(&blk.EVMBlock)
		for i := range blk.Transactions {
			st.ExecuteTransaction(&blk.Transactions[i])
		}
		st.EndBlock(blk.UncleBlocks)
		assert.EQ(st.Commit().Hex(), blk.StateRoot.Hex())
		progress_bar.Add(1)
	}
}

func mkdirp(path string) string {
	util.PanicIfNotNil(exec.Command("mkdir", "-p", path).Run())
	return path
}
