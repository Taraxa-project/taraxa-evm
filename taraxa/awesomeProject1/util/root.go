package util

import (
	"encoding/binary"
	"fmt"
)

func Assert(cond bool) {
	if !cond {
		panic("")
	}
}

func PanicIfNotNil(err error) {
	if err != nil {
		panic(err)
	}
}

var compact_table_64 = func() (ret [8]uint64) {
	for i := 0; i < len(ret); i++ {
		ret[i] = 1<<((i+1)*8) - 1
	}
	return
}()

func ENC_b_endian_compact_64_w_alloc(n uint64, alloc func(int) []byte) []byte {
	for i := 0; i < 8; i++ {
		if n <= compact_table_64[i] {
			size := i + 1
			ret := alloc(size)
			for j := 0; j < size; j++ {
				ret = append(ret, byte(n>>(8*(i-j))))
			}
			return ret
		}
	}
	panic("")
}

func ENC_b_endian_compact_64(n uint64) []byte {
	return ENC_b_endian_compact_64_w_alloc(n, func(size int) []byte {
		return make([]byte, 0, size)
	})
}

func DEC_b_endian_compact_64(enc []byte) (ret uint64) {
	for magnitude, last_pos := 0, len(enc)-1; magnitude <= last_pos; magnitude++ {
		ret |= uint64(enc[last_pos-magnitude]) << (8 * magnitude)
	}
	return ret
}

func ENC_b_endian_64(n uint64) []byte {
	ret := make([]byte, 8)
	binary.BigEndian.PutUint64(ret, n)
	return ret
}

func BytesToStrPadded(bytes []byte) (ret string) {
	for i, b := range bytes {
		ret += fmt.Sprintf("%03d", b)
		if i < len(bytes)-1 {
			ret += " "
		}
	}
	return ret
}