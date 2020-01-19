package cgo

//#include <stdlib.h>
//#include "index.h"
import "C"
import "unsafe"

// TODO optimize

func slice(b []byte) C.taraxa_cgo_ethdb_Slice {
	return C.taraxa_cgo_ethdb_Slice_New(
		(*C.taraxa_cgo_ethdb_SliceType)(C.CBytes(b)),
		C.taraxa_cgo_ethdb_SliceSize(len(b)),
	)
}

func str(s C.taraxa_cgo_ethdb_Slice) string {
	return C.GoStringN(s.offset, s.size)
}

func bytes(s C.taraxa_cgo_ethdb_Slice) []byte {
	return []byte(str(s))
}

func free(s C.taraxa_cgo_ethdb_Slice) {
	C.free((unsafe.Pointer)(s.offset))
}
