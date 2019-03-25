package main

import "C"
import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/taraxa_vm/util"
)

//export Run
func Run(input string) string {
	runConfig := new(RunConfig)
	err := json.Unmarshal([]byte(input), runConfig)
	util.FailOnErr(err)
	result, err := Process(runConfig)
	util.FailOnErr(err)
	bytes, err := json.Marshal(&result)
	util.FailOnErr(err)
	return string(bytes)
}

func main() {

}
