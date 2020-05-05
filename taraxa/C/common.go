package main

//#include "common.h"
import "C"
import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"reflect"
	"runtime/debug"
	"unsafe"
)

func dec_rlp(enc C.taraxa_evm_Bytes, out interface{}) {
	rlp.MustDecodeBytes(bin.AnyBytes2(unsafe.Pointer(enc.Data), int(enc.Len)), out)
}

func dec_json(enc C.taraxa_evm_Bytes, out interface{}) {
	util.PanicIfNotNil(json.Unmarshal(bin.AnyBytes2(unsafe.Pointer(enc.Data), int(enc.Len)), out))
}

func bytes_to_c(b []byte) (ret C.taraxa_evm_Bytes) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	ret.Data = (*C.uint8_t)(unsafe.Pointer(sh.Data))
	ret.Len = C.size_t(sh.Len)
	return
}

func call_bytes_cb(b []byte, cb C.taraxa_evm_BytesCallback) {
	C.taraxa_evm_BytesCallbackApply(cb, bytes_to_c(b))
}

func enc_rlp(in interface{}, out C.taraxa_evm_BytesCallback) {
	call_bytes_cb(rlp.MustEncodeToBytes(in), out)
}

func handle_err(cb C.taraxa_evm_BytesCallback) {
	if issue := recover(); issue != nil {
		msg := "\n" +
			"=== Error. Message:\n" +
			fmt.Sprint(issue) + "\n" +
			"=== backtrace:\n" +
			bin.StringView(debug.Stack())
		call_bytes_cb(bin.BytesView(msg), cb)
	}
}
