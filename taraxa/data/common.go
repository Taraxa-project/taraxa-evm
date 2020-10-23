package data

import (
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/files"
	"os"
)

var this_dir = files.ThisDirRelPath()

func parse_rlp_file(short_file_name string, out interface{}) {
	f, err := os.Open(files.Path(this_dir, short_file_name+".rlp"))
	util.PanicIfNotNil(err)
	util.PanicIfNotNil(rlp.Decode(f, out))
}
