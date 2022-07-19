package bin

import (
	"math/rand"
	"reflect"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

func StringView(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&reflect.StringHeader{
		(*reflect.SliceHeader)(unsafe.Pointer(&bytes)).Data,
		len(bytes),
	}))
}

func BytesView(str string) (ret []byte) {
	ret = AnyBytes1((*reflect.StringHeader)(unsafe.Pointer(&str)).Data, len(str))
	return
}

func AnyBytes1(ptr uintptr, length int) (ret []byte) {
	ret = *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{ptr, length, length}))
	return
}

func AnyBytes2(ptr unsafe.Pointer, length int) (ret []byte) {
	ret = AnyBytes1(uintptr(ptr), length)
	return
}

func Concat(s1 []byte, s2 ...byte) (ret []byte) {
	ret = make([]byte, len(s1)+len(s2))
	copy(ret[copy(ret, s1):], s2)
	return
}

func Concat2(bs ...[]byte) (ret []byte) {
	l := 0
	for _, b := range bs {
		l += len(b)
	}
	ret = make([]byte, 0, l)
	for _, b := range bs {
		ret = append(ret, b...)
	}
	return
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

func ENC_b_endian_compact_64_1(i uint64) (ret []byte) {
	ENC_b_endian_compact_64(i, func(b ...byte) {
		ret = b
	})
	return
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
	default:
		panic("impossible")
	}
}

func SizeInBytes(i uint64) byte {
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

func RandomBytes(desired_len int, rnd *rand.Rand) []byte {
	ret := make([]byte, 0, desired_len)
	for {
		curr_len := len(ret)
		if curr_len == desired_len {
			break
		}
		var i uint64
		if rnd == nil {
			i = rand.Uint64()
		} else {
			i = rnd.Uint64()
		}
		ENC_b_endian_compact_64(i, func(b ...byte) {
			ret = append(ret, b[:util.Min(len(b), desired_len-curr_len)]...)
		})
	}
	return ret
}

const zfill_batch_size = 8 * 1024

var zfill_batch = make([]byte, zfill_batch_size)

func ZFill_1(s []byte) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	for zfill_batch_size <= sh.Len {
		*(*[zfill_batch_size]byte)(unsafe.Pointer(sh.Data)) = [zfill_batch_size]byte{}
		sh.Data += zfill_batch_size
		sh.Len -= zfill_batch_size
	}
	if sh.Len != 0 {
		copy(s, zfill_batch)
	}
}

func ZFill_2(slice_ptr unsafe.Pointer, max_elements int, elem_size uintptr) {
	sh := *(*reflect.SliceHeader)(slice_ptr)
	if max_elements < 0 {
		sh.Len = sh.Len * int(elem_size)
	} else {
		sh.Len = util.Min(sh.Len, max_elements) * int(elem_size)
	}
	ZFill_1(*(*[]byte)(unsafe.Pointer(&sh)))
}

func ZFill_3(slice_ptr unsafe.Pointer, elem_size uintptr) {
	ZFill_2(slice_ptr, -1, elem_size)
}

func UnsafeAdd(p unsafe.Pointer, x uintptr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

var IsPlatformBigEndian = func() bool {
	var buf [2]byte
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xabcd)
	switch buf {
	case [2]byte{0xcd, 0xab}:
		return false
	case [2]byte{0xab, 0xcd}:
		return true
	default:
		panic("Could not determine native endianness.")
	}
}()
