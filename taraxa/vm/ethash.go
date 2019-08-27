package vm

import (
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
)

func AccumulateMiningRewards(
	config *params.ChainConfig,
	state *state.StateDB,
	header *BlockNumberAndCoinbase,
	uncles ...*BlockNumberAndCoinbase,
) {
	var unclesMapped []*types.Header
	for _, uncle := range uncles {
		unclesMapped = append(unclesMapped, &types.Header{Number: uncle.Number, Coinbase: uncle.Coinbase})
	}
	ethash.AccumulateRewards(
		config,
		state,
		&types.Header{Number: header.Number, Coinbase: header.Coinbase},
		unclesMapped)
}
