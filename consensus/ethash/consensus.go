// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

var (
	CalifornicumBlockReward = big.NewInt(2e+18) // Block reward in wei for successfully mining a block upward from CalifornicumBlockReward
)

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

type BlockNumAndCoinbase = struct {
	Number types.BlockNum
	Author common.Address
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func AccumulateRewards(
	// rules vm.Rules,
	header BlockNumAndCoinbase,
	uncles []BlockNumAndCoinbase,
	state vm.State) {
	// Select the correct block reward based on chain progression
	blockReward := CalifornicumBlockReward
	// Accumulate the rewards for the miner and any included uncles
	header_num_big := new(big.Int).SetUint64(header.Number)
	reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.SetUint64(uncle.Number + 8)
		r.Sub(r, header_num_big)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		state.GetAccount(&uncle.Author).AddBalance(r)
		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}
	state.GetAccount(&header.Author).AddBalance(reward)
}
