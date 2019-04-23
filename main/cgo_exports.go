package main

//#cgo CFLAGS: -I ../lib_cpp/include
//#include "taraxa_evm/cgo_imports.h"
import "C"

import (
	"github.com/Taraxa-project/taraxa-evm/main/api/facade"
)

//export RunTaraxaEvm
func RunTaraxaEvm(input *C.char) *C.char {
	response := facade.RunJson(C.GoString(input))
	return C.CString(response)
}

func main() {

}
