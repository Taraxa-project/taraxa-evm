//+build extern

package external

//#include <stdlib.h>
//#include "taraxa_vm_external.h"
import "C"
import "github.com/Taraxa-project/taraxa-evm/common"

//import ( TODO
//	"fmt"
//	"unsafe"
//)

func init() {
	GetHeaderHashByBlockNumber = func(u uint64) common.Hash {
		// TODO call C functions
	}
}