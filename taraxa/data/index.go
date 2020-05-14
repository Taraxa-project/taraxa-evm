package data

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"os"
	"path"
	"runtime"
)

var Dir = func() string {
	_, this_file, _, _ := runtime.Caller(0)
	return path.Dir(this_file)
}()

func Parse_eth_mainnet_genesis_accounts() (ret state_transition.AccountMap) {
	f, err := os.Open(path.Join(Dir, "eth_mainnet_genesis_accounts.rlp"))
	util.PanicIfNotNil(err)
	util.PanicIfNotNil(rlp.Decode(f, &ret))
	return
}

func Parse_eth_mainnet_blocks_0_300000() (ret []struct {
	Hash         common.Hash
	StateRoot    common.Hash
	EVMBlock     vm.BlockWithoutNumber
	Transactions []vm.Transaction
	UncleBlocks  []state_transition.UncleBlock
}) {
	f, err := os.Open(path.Join(Dir, "eth_mainnet_blocks_0_300000.rlp"))
	util.PanicIfNotNil(err)
	util.PanicIfNotNil(rlp.Decode(f, &ret))
	return
}
