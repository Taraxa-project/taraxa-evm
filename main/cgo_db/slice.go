package cgo_db

//#include "../cgo_imports.h"
import "C"
import "unsafe"

func slice(b []byte) C.taraxa_cgo_ethdb_Slice {
	return C.taraxa_cgo_ethdb_Slice_New(
		(*C.taraxa_cgo_ethdb_SliceType)(C.CBytes(b)),
		C.taraxa_cgo_ethdb_SliceSize(len(b)),
	);
}

func bytes(s C.taraxa_cgo_ethdb_Slice) []byte {
	return C.GoBytes(unsafe.Pointer(s.offset), C.int(s.size));
}

func free(s C.taraxa_cgo_ethdb_Slice) {
	C.free(unsafe.Pointer(s.offset))
}
