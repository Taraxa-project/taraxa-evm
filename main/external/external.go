//TODO +build lib_cpp

package external

import "C"
import "github.com/Taraxa-project/taraxa-evm/common"

//import ( TODO
//	"fmt"
//	"unsafe"
//)

func init() {
	GetHeaderHashByBlockNumber = func(u uint64) common.Hash {
		return common.Hash{}
		// TODO call C functions
	}
}