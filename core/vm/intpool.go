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
	"sync"
)

const bigint_pool_limit = 512

var poolOfIntPools = intPoolPool{sync.Pool{New: func() interface{} {
	return &int_pool{pool: stack{data: make([]*big.Int, 0, bigint_pool_limit)}}
}}}

// intPoolPool manages a pool of intPools.
type intPoolPool struct {
	pool sync.Pool
}

// get is looking for an available pool to return.
func (this *intPoolPool) get() *int_pool {
	return this.pool.Get().(*int_pool)
}

// put a pool that has been allocated with get.
func (this *intPoolPool) put(ip *int_pool) {
	this.pool.Put(ip)
}

// int_pool is a pool of big integers that
// can be reused for all big.Int operations.
type int_pool struct {
	pool stack
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
		return p.pool.pop().SetUint64(0)
	}
	return new(big.Int)
}

// put returns an allocated big int to the pool to be later reused by get calls.
// Note, the values as saved as is; neither put nor get zeroes the ints out!
func (p *int_pool) put(is ...*big.Int) {
	for _, i := range is {
		if len(p.pool.data) == bigint_pool_limit {
			return
		}
		p.pool.push(i)
	}
}
