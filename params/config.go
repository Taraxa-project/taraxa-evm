// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

var (
	MainnetChainConfig = &ChainConfig{
		ChainId: 841,
	}
)

type ChainConfig struct {
	ChainId uint64 `json:"chainId"`
	// HomesteadBlock types.BlockNum `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

// func (c *ChainConfig) Rules(num types.BlockNum) vm.Rules {
// 	return vm.Rules{
// 		IsHomestead:      isForked(c.HomesteadBlock, num),
// 		IsEIP150:         isForked(c.EIP150Block, num),
// 		IsEIP158:         isForked(c.EIP158Block, num),
// 		IsByzantium:      isForked(c.ByzantiumBlock, num),
// 		IsConstantinople: isForked(c.ConstantinopleBlock, num),
// 		IsPetersburg:     isForked(c.PetersburgBlock, num) || c.PetersburgBlock == types.BlockNumberNIL && isForked(c.ConstantinopleBlock, num),
// 	}
// }
