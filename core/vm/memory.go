package vm

import (
	"fmt"
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/common/math"
)

type MemoryPool struct {
	buf      []byte
	reserved int
}

func (self *MemoryPool) Init(capacity uint64) *MemoryPool {
	self.buf = make([]byte, capacity)
	return self
}

// Memory implements a simple memory model for the ethereum virtual machine.
type Memory struct {
	pool        *MemoryPool
	pool_offset int
	using_pool  bool
	store       []byte
	lastGasCost uint64
}

func (self *Memory) Init(pool *MemoryPool) *Memory {
	self.pool = pool
	self.pool_offset = pool.reserved
	self.using_pool = true
	return self
}

func (self *Memory) Release() {
	self.release(nil)
}

func (self *Memory) release(move_to []byte) {
	copy(move_to, self.store)
	if self.using_pool {
		self.using_pool = false
		bin.ZFill_1(self.store)
		self.pool.reserved = self.pool_offset
	}
	self.store = move_to
}

// Set sets offset + size to value
func (m *Memory) Set(offset, size uint64, value []byte) {
	// It's possible the offset is greater than 0 and size equals 0. This is because
	// the calcMemSize (common.go) could potentially return 0 when size is zero (NO-OP)
	if size > 0 {
		// length of store may never be less than offset + size.
		// The store should be resized PRIOR to setting the memory
		if offset+size > m.Len() {
			panic("invalid memory: store empty")
		}
		copy(m.store[offset:offset+size], value)
	}
}

// Set32 sets the 32 bytes starting at offset to the value of val, left-padded with zeroes to
// 32 bytes.
func (m *Memory) Set32(offset uint64, val *big.Int) {
	// length of store may never be less than offset + size.
	// The store should be resized PRIOR to setting the memory
	end := offset + 32
	if end > m.Len() {
		panic("invalid memory: store empty")
	}
	chunk := m.store[offset:end]
	// Fill in relevant bits, zero the memory area
	bin.ZFill_1(chunk[:math.ReadBits(val, chunk)])
}

// Resize resizes the memory to size
func (m *Memory) Resize(size uint64) {
	if size <= m.Len() {
		return
	}
	if m.using_pool {
		pool_reserved := m.pool_offset + int(size)
		if pool_reserved <= len(m.pool.buf) {
			m.pool.reserved = pool_reserved
			m.store = m.pool.buf[m.pool_offset:pool_reserved]
			return
		}
	}
	m.release(make([]byte, size))
}

// Get returns offset + size as a new slice
func (m *Memory) Get(offset, size int64) (cpy []byte) {
	if size != 0 && len(m.store) > int(offset) {
		cpy = make([]byte, size)
		copy(cpy, m.store[offset:offset+size])
	}
	return
}

// Len returns the length of the backing slice
func (m *Memory) Len() uint64 {
	return uint64(len(m.store))
}

// Print dumps the content of the memory.
func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.store))
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			fmt.Printf("%03d: % x\n", addr, m.store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
