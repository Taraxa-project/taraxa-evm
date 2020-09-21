package data

import (
	"os"
	"path"
	"runtime"

	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

var this_dir = func() string {
	_, this_file, _, _ := runtime.Caller(0)
	return path.Dir(this_file)
}()

func parse_rlp_file(short_file_name string, out interface{}) {
	f, err := os.Open(path.Join(this_dir, short_file_name+".rlp"))
	util.PanicIfNotNil(err)
	util.PanicIfNotNil(rlp.Decode(f, out))
}
