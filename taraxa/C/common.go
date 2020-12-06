package main

//#include "common.h"
import "C"
import (
	"fmt"
	"path"
	"reflect"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

func dec_rlp(enc C.taraxa_evm_Bytes, out interface{}) {
	rlp.MustDecodeBytes(c_bytes_to_go(enc), out)
}

func go_bytes_to_c(b []byte) (ret C.taraxa_evm_Bytes) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	ret.Data = (*C.uint8_t)(unsafe.Pointer(sh.Data))
	ret.Len = C.size_t(sh.Len)
	return
}

func c_bytes_to_go(b C.taraxa_evm_Bytes) []byte {
	return bin.AnyBytes2(unsafe.Pointer(b.Data), int(b.Len))
}

func call_bytes_cb(b []byte, cb C.taraxa_evm_BytesCallback) {
	C.taraxa_evm_BytesCallbackApply(cb, go_bytes_to_c(b))
}

func enc_rlp(in interface{}, out C.taraxa_evm_BytesCallback) {
	call_bytes_cb(rlp.MustEncodeToBytes(in), out)
}

func handle_err(cb C.taraxa_evm_BytesCallback) {
	if issue := recover(); issue != nil {
		typ := reflect.TypeOf(issue)
		call_bytes_cb(bin.BytesView(path.Join(typ.PkgPath(), typ.Name())+": "+fmt.Sprint(issue)), cb)
	}
}
