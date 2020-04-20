package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
)

type TxIndex = uint

type Account struct {
	Nonce           uint64
	Balance         *big.Int
	StorageRootHash *common.Hash
	CodeHash        *common.Hash
	CodeSize        uint64
}

type ChainConfig struct {
	EvmChainConfig
	DisableBlockRewards bool
}

type EvmChainConfig struct {
	EthChainCfg      params.ChainConfig
	EvmExecutionOpts vm.ExecutionOptions
}
