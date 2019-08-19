package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/taraxa_types"
)

func AccumulateRewards(config *params.ChainConfig, state *state.StateDB, header *taraxa_types.BlockNumberAndCoinbase,
	uncles ...*taraxa_types.BlockNumberAndCoinbase) {
	var unclesMapped []*types.Header
	for _, uncle := range uncles {
		unclesMapped = append(unclesMapped, &types.Header{Number: uncle.Number, Coinbase: uncle.Coinbase})
	}
	ethash.AccumulateRewards(config, state, &types.Header{Number: header.Number, Coinbase: header.Coinbase}, unclesMapped)
}
