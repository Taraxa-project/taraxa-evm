package jsonutil

import (
	"encoding/json"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

func MustEncode(obj interface{}) []byte {
	bs, err := json.Marshal(obj)
	util.PanicIfNotNil(err)
	return bs
}

func MustEncodePretty(obj interface{}) []byte {
	bs, err := json.MarshalIndent(obj, "", "    ")
	util.PanicIfNotNil(err)
	return bs
}

func MustDecode(b []byte, obj interface{}) {
	util.PanicIfNotNil(json.Unmarshal(b, obj))
}
