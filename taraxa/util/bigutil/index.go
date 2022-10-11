package bigutil

import (
	"bytes"
	"math"
	"math/big"
	"reflect"
	"unsafe"
)

const WordSize = int(unsafe.Sizeof(big.Word(0)))

var MaxU256 = new(big.Int).SetBytes(bytes.Repeat([]byte{math.MaxUint8}, 32))

func FromByte(b byte) *big.Int {
	return big.NewInt(int64(b))
}

func FromBytes(bytes []byte) *big.Int {
	switch len(bytes) {
	case 0:
		return big.NewInt(0)
	case 1:
		return big.NewInt(int64(bytes[0]))
	default:
		return new(big.Int).SetBytes(bytes)
	}
}

func IsZero(x *big.Int) bool {
	return x == nil || x.Sign() == 0
}

func ZeroIfNIL(x *big.Int) *big.Int {
	if x == nil {
		return big.NewInt(0)
	}
	return x
}

func Add(x, y *big.Int) *big.Int {
	return new(big.Int).Add(ZeroIfNIL(x), ZeroIfNIL(y))
}

func Sub(x, y *big.Int) *big.Int {
	return new(big.Int).Sub(ZeroIfNIL(x), ZeroIfNIL(y))
}

func Div(x, y *big.Int) *big.Int {
	return new(big.Int).Div(ZeroIfNIL(x), ZeroIfNIL(y))
}

func Mul(x, y *big.Int) *big.Int {
	return new(big.Int).Mul(ZeroIfNIL(x), ZeroIfNIL(y))
}

type UnsignedBytes []byte
type UnsignedStr string

func (self UnsignedStr) Int() *big.Int {
	h := (*reflect.StringHeader)(unsafe.Pointer(&self))
	if h.Len == 0 {
		return big.NewInt(0)
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
