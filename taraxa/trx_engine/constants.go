package trx_engine

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
)

var TaraxaGenesisConfig = &core.Genesis{
	// TODO consume from taraxa_node
	Config: &params.ChainConfig{
		ChainID:             big.NewInt(66),
		HomesteadBlock:      common.Big0,
		EIP150Block:         common.Big0,
		EIP155Block:         common.Big0,
		EIP158Block:         common.Big0,
		ByzantiumBlock:      common.Big0,
		ConstantinopleBlock: common.Big0,
		//PetersburgBlock:     common.Big0,
	},
}
