package binary

import (
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

func Concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}
