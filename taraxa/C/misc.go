package main

//#include "common.h"
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

func main() {

}
