package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

type UncleBlock = ethash.BlockNumAndCoinbase

var EmptyRLPListHash = func() common.Hash {
	return crypto.Keccak256Hash(rlp.MustEncodeToBytes([]byte(nil)))
}()

func IsEmptyStateRoot(state_root *common.Hash) bool {
	return state_root == nil || *state_root == EmptyRLPListHash || *state_root == common.ZeroHash
}
