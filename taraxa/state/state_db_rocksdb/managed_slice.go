package state_db_rocksdb

import "github.com/tecbot/gorocksdb"

type managed_slice struct {
	h *gorocksdb.PinnableSliceHandle
	v []byte
}

func (self *managed_slice) Init(h *gorocksdb.PinnableSliceHandle) *managed_slice {
	self.h = h
	self.v = h.Data()
	return self
}

func (self *managed_slice) Value() []byte {
	return self.v
}

func (self *managed_slice) Free() {
	self.h.Destroy()
}
