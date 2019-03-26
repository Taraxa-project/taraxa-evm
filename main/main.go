//+build lib_cpp

package main

import "C"
import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

//export RunEvm
func RunEvm(input *C.char) *C.char {
	runConfig := new(RunConfiguration)
	err := json.Unmarshal([]byte(C.GoString(input)), runConfig)
	util.FailOnErr(err)
	result, _ := Process(runConfig)
	bytes, err := json.Marshal(&result)
	util.FailOnErr(err)
	return C.CString(string(bytes))
}

func main() {

}
