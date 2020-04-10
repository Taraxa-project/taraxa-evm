package bin

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"reflect"
	"unsafe"
)

func StringView(bytes []byte) string {
	h := (*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{h.Data, h.Len}))
}

func BytesView(str string) []byte {
	h := (*reflect.StringHeader)(unsafe.Pointer(&str))
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{h.Data, h.Len, h.Len}))
}

func HashView(bytes []byte) (ret *common.Hash) {
	if l := len(bytes); l != 0 && assert.Holds(l == common.HashLength) {
		ret = (*common.Hash)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&bytes)).Data))
	}
	return
}

func AddrView(bytes []byte) *common.Address {
	return (*common.Address)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&bytes)).Data))
}

func Concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}

func ENC_b_endian_64(v uint64) []byte {
	return []byte{
		byte(v >> 56),
		byte(v >> 48),
		byte(v >> 40),
		byte(v >> 32),
		byte(v >> 24),
		byte(v >> 16),
		byte(v >> 8),
		byte(v),
	}
}

func DEC_b_endian_64(b []byte) uint64 {
	return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
		uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
}

func ENC_b_endian_compact_64(i uint64, appender func(...byte)) {
	switch {
	case i < (1 << 8):
		appender(byte(i))
	case i < (1 << 16):
		appender(byte(i>>8), byte(i))
	case i < (1 << 24):
		appender(byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 32):
		appender(byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 40):
		appender(byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 48):
		appender(byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	case i < (1 << 56):
		appender(byte(i>>48), byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	default:
		appender(byte(i>>56), byte(i>>48), byte(i>>40), byte(i>>32), byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
	}
}

func DEC_b_endian_compact_64(b []byte) uint64 {
	switch len(b) {
	case 0:
		return 0
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(b[0])<<8 | uint64(b[1])
	case 3:
		return uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		return uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		return uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 |
			uint64(b[4])
	case 6:
		return uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 |
			uint64(b[4])<<8 | uint64(b[5])
	case 7:
		return uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 |
			uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	panic("impossible")
}

func ActualSizeInBytes(i uint64) int {
	switch {
	case i < (1 << 8):
		return 1
	case i < (1 << 16):
		return 2
	case i < (1 << 24):
		return 3
	case i < (1 << 32):
		return 4
	case i < (1 << 40):
		return 5
	case i < (1 << 48):
		return 6
	case i < (1 << 56):
		return 7
	default:
		return 8
	}
}
