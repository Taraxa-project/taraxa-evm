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
