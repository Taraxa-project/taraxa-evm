package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
)

type TxIndex = uint32

type Account struct {
	Nonce           uint64
	Balance         *big.Int
	StorageRootHash *common.Hash
	CodeHash        *common.Hash
	CodeSize        uint64
}

type ChainConfig struct {
	EVMChainConfig
	DisableBlockRewards bool
}
type EVMChainConfig struct {
	ETHChainConfig
	vm.ExecutionOptions
}
type ETHChainConfig = params.ChainConfig
