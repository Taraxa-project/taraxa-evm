package util

import (
	"sync/atomic"
	"unsafe"
)

type ErrorBarrier struct {
	err unsafe.Pointer
}

func (this *ErrorBarrier) SetIfAbsent(err error) (hasSet bool) {
	if err != nil {
		hasSet = atomic.CompareAndSwapPointer(&this.err, nil, unsafe.Pointer(&err))
	}
	return
}

func (this *ErrorBarrier) CheckIn(errors ...error) {
	this.PanicIfPresent()
	for _, err := range errors {
		if err == nil {
			continue
		}
		this.SetIfAbsent(err)
		this.PanicIfPresent()
	}
}

func (this *ErrorBarrier) PanicIfPresent() {
	if loaded := atomic.LoadPointer(&this.err); loaded != nil {
		errPtr := (*error)(loaded)
		panic(*errPtr)
	}
}

func (this *ErrorBarrier) Recover(callbacks ...func(error)) {
	if recovered := recover(); recovered != nil {
		loaded := atomic.LoadPointer(&this.err)
		thisErrPtr := (*error)(loaded)
		if thisErrPtr != nil && recovered == *thisErrPtr {
			err := recovered.(error)
			for _, cb := range callbacks {
				cb(err)
			}
		} else {
			panic(recovered)
		}
	}
}
