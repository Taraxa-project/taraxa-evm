//+build lib_cpp

package main

//#cgo CFLAGS: -I ../lib_cpp/include
//#include "taraxa_evm/cgo_imports.h"
import "C"

import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/state_transition"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

//export RunEvm
func RunEvm(input *C.char, externalApi *C.ExternalApi) *C.char {
	runConfig := new(api.RunConfiguration)
	err := json.Unmarshal([]byte(C.GoString(input)), runConfig)
	util.PanicOn(err)
	result, _ := state_transition.Run(runConfig, &api.ExternalApi{
		GetHeaderHashByBlockNumber: func(n uint64) common.Hash {
			c_str := C.getHeaderHashByBlockNumber(externalApi, C.uint64_t(n))
			return common.HexToHash(C.GoString(c_str))
		},
	})
	bytes, err := json.Marshal(&result)
	util.PanicOn(err)
	return C.CString(string(bytes))
}

func main() {

}
