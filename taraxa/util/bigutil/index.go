package bigutil

import (
	"bytes"
	"math"
	"math/big"
	"reflect"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
)

const WordSize = int(unsafe.Sizeof(big.Word(0)))

var cache = func() (ret [256 + 1]*big.Int) {
	for i := range ret {
		ret[i] = big.NewInt(int64(i))
	}
	return
}()
var Big0 = cache[0]
var Big1 = cache[1]
var Big32 = cache[32]
var Big256 = cache[256]
var MaxU256 = new(big.Int).SetBytes(bytes.Repeat([]byte{math.MaxUint8}, 32))

func FromByte(b byte) *big.Int {
	return cache[b]
}

func FromBytes(bytes []byte) *big.Int {
	switch len(bytes) {
	case 0:
		return Big0
	case 1:
		return cache[bytes[0]]
	default:
		return new(big.Int).SetBytes(bytes)
	}
}

func ZeroIfNIL(x *big.Int) *big.Int {
	if x == nil {
		return Big0
	}
	return x
}

func Add(x, y *big.Int) *big.Int {
	if x == nil || x.Sign() == 0 {
		return ZeroIfNIL(y)
	}
	if y == nil || y.Sign() == 0 {
		return ZeroIfNIL(x)
	}
	return new(big.Int).Add(x, y)
}

func USub(x, y *big.Int) *big.Int {
	if x == nil || x.Sign() == 0 {
		ret := ZeroIfNIL(y)
		asserts.Holds(ret.Sign() == 0)
		return ret
	}
	if y == nil || y.Sign() == 0 {
		return ZeroIfNIL(x)
	}
	return new(big.Int).Sub(x, y)
}

type UnsignedBytes []byte
type UnsignedStr string

func (self UnsignedStr) Int() *big.Int {
	h := (*reflect.StringHeader)(unsafe.Pointer(&self))
	if h.Len == 0 {
		return Big0
	}
	l := h.Len / WordSize
	return new(big.Int).SetBits(*(*[]big.Word)(unsafe.Pointer(&reflect.SliceHeader{h.Data, l, l})))
}

func UnsafeUnsignedBytes(i *big.Int) (ret UnsignedBytes) {
	bits := i.Bits()
	h := (*reflect.SliceHeader)(unsafe.Pointer(&bits))
	if h.Len != 0 {
		l := h.Len * WordSize
		ret = *(*UnsignedBytes)(unsafe.Pointer(&reflect.SliceHeader{h.Data, l, l}))
	}
	return
}
