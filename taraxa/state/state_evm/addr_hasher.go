package state_evm

import (
	"crypto/rand"
	"math"
	"math/big"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"

	"github.com/Taraxa-project/taraxa-evm/common"
)

var hashkey = func() (ret [4]uintptr) {
	for i := range ret {
		ret[i] = rand_uintptr() | 1
	}
	return
}()

func rand_uintptr() uintptr {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(math.MaxInt64)))
	if err != nil {
		panic(err)
	}
	return uintptr(nBig.Uint64() % uint64(^uintptr(0)))
}

func hash_addr(a *common.Address, seed uintptr) uintptr {
	const (
		size = common.AddressLength
		m1   = 16877499708836156737
		m2   = 2820277070424839065
		m3   = 9497967016996688599
	)
	p := unsafe.Pointer(a)
	h := uint64(seed + size*hashkey[0])
	h ^= read_mem_64(p)
	h = rotl_31(h*m1) * m2
	h ^= read_mem_64(bin.UnsafeAdd(p, 8))
	h = rotl_31(h*m1) * m2
	h ^= read_mem_64(bin.UnsafeAdd(p, size-16))
	h = rotl_31(h*m1) * m2
	h ^= read_mem_64(bin.UnsafeAdd(p, size-8))
	h = rotl_31(h*m1) * m2
	h ^= h >> 29
	h *= m3
	h ^= h >> 32
	return uintptr(h)
}

func rotl_31(x uint64) uint64 {
	return (x << 31) | (x >> (64 - 31))
}

var read_mem_64 = func() func(unsafe.Pointer) uint64 {
	if bin.IsPlatformBigEndian {
		return func(p unsafe.Pointer) uint64 {
			q := (*[8]byte)(p)
			return uint64(q[7]) | uint64(q[6])<<8 | uint64(q[5])<<16 | uint64(q[4])<<24 |
				uint64(q[3])<<32 | uint64(q[2])<<40 | uint64(q[1])<<48 | uint64(q[0])<<56
		}
	}
	return func(p unsafe.Pointer) uint64 {
		q := (*[8]byte)(p)
		return uint64(q[0]) | uint64(q[1])<<8 | uint64(q[2])<<16 | uint64(q[3])<<24 |
			uint64(q[4])<<32 | uint64(q[5])<<40 | uint64(q[6])<<48 | uint64(q[7])<<56
	}
}()
