package main

//#include "types.h"
import "C"
import (
	"runtime"
	"runtime/debug"
)

//export taraxa_evm_SetGCPercent
func taraxa_evm_SetGCPercent(pct C.int) {
	debug.SetGCPercent(int(pct))
}

//export taraxa_evm_GC
func taraxa_evm_GC() {
	runtime.GC()
}

//export taraxa_evm_Traceback
func taraxa_evm_Traceback(cb C.taraxa_evm_BytesCallback) {
	call_bytes_cb(debug.Stack(), cb)
}

func main() {

}
