package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"
)

type ChainConfig struct {
	Execution ExecutionConfig
	DPOS      *dpos.Config
}
type ExecutionConfig struct {
	DisableBlockRewards bool
	ETHForks            params.ChainConfig
	Options             vm.ExecutionOptions
}

type UncleBlock = ethash.BlockNumAndCoinbase

var EmptyRLPListHash = func() common.Hash {
	return crypto.Keccak256Hash(rlp.MustEncodeToBytes([]byte(nil)))
}()
