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
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

var (
	MainnetChainConfig = &ChainConfig{
		HomesteadBlock:      1150000,
		EIP150Block:         2463000,
		EIP158Block:         2675000,
		ByzantiumBlock:      4370000,
		ConstantinopleBlock: 7280000,
		PetersburgBlock:     7280000,
	}
)

type ChainConfig struct {
	HomesteadBlock types.BlockNum `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)
	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block         types.BlockNum `json:"eip150Block,omitempty"`         // EIP150 HF block (nil = no fork)
	EIP158Block         types.BlockNum `json:"eip158Block,omitempty"`         // EIP158 HF block
	ByzantiumBlock      types.BlockNum `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock types.BlockNum `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PetersburgBlock     types.BlockNum `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
}

func isForked(fork_start, block_num types.BlockNum) bool {
	if fork_start == types.BlockNumberNIL || block_num == types.BlockNumberNIL {
		return false
	}
	return fork_start <= block_num
}

func (c *ChainConfig) Rules(num types.BlockNum) vm.Rules {
	return vm.Rules{
		IsHomestead:      isForked(c.HomesteadBlock, num),
		IsEIP150:         isForked(c.EIP150Block, num),
		IsEIP158:         isForked(c.EIP158Block, num),
		IsByzantium:      isForked(c.ByzantiumBlock, num),
		IsConstantinople: isForked(c.ConstantinopleBlock, num),
		IsPetersburg:     isForked(c.PetersburgBlock, num) || c.PetersburgBlock == types.BlockNumberNIL && isForked(c.ConstantinopleBlock, num),
	}
}
