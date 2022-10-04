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

package vm

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
)

// int_pool is a pool of big integers that
// can be reused for all big.Int operations.
type int_pool struct {
	pool Stack
}

func (self *int_pool) Init(capacity int) *int_pool {
	self.pool.Init(capacity)
	for i := 0; i < capacity; i++ {
		self.pool.push(new(big.Int).Set(bigutil.MaxU256))
	}
	return self
}

// get retrieves a big int from the pool, allocating one if the pool is empty.
// Note, the returned int's value is arbitrary and will not be zeroed!
func (p *int_pool) get() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop()
	}
	return new(big.Int)
}

// getZero retrieves a big int from the pool, setting it to zero or allocating
// a new one if the pool is empty.
func (p *int_pool) getZero() *big.Int {
	if p.pool.len() > 0 {
		return p.pool.pop().Set(big.NewInt(0))
	}
	return new(big.Int)
}

// put returns an allocated big int to the pool to be later reused by get calls.
// Note, the values as saved as is; neither put nor get zeroes the ints out!
func (p *int_pool) put(is ...*big.Int) {
	for _, i := range is {
		if len(p.pool.data) == cap(p.pool.data) {
			return
		}
		p.pool.push(i)
	}
}
