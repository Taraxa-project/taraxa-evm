package data

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
)

func Parse_eth_mainnet_blocks_0_300000() (ret []struct {
	Hash         common.Hash
	StateRoot    common.Hash
	EVMBlock     vm.BlockInfo
	Transactions []vm.Transaction
	UncleBlocks  []state_common.UncleBlock
}) {
	parse_rlp_file("eth_mainnet_blocks_0_300000", &ret)
	return
}
