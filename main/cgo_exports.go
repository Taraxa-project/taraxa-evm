package main

//#cgo CFLAGS: -I ../lib_cpp/include
//#include "taraxa_evm/cgo_imports.h"
import "C"

import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/api/facade"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

//export RunTaraxaEvm
func RunTaraxaEvm(input *C.char) *C.char {
	request := new(api.Request)
	err := json.Unmarshal([]byte(C.GoString(input)), request)
	util.PanicOn(err)
	response := facade.Run(request)
	bytes, err := json.Marshal(&response)
	util.PanicOn(err)
	return C.CString(string(bytes))
}

func main() {

}
