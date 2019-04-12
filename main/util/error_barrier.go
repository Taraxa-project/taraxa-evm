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
	PanicOn(this.Get())
}

func (this *ErrorBarrier) Get() error {
	ptr := (*error)(atomic.LoadPointer(&this.err))
	if ptr != nil {
		return *ptr
	}
	return nil
}

func (this *ErrorBarrier) Recover(callbacks ...func(error)) {
	if recovered := recover(); recovered != nil {
		thisErr := this.Get()
		if recovered == thisErr {
			for _, callback := range callbacks {
				callback(thisErr)
			}
		} else {
			panic(recovered)
		}
	}
}

func (this *ErrorBarrier) Catch(handlers ...func(err error)) Hanlder {
	return func(caught interface{}) bool {
		thisErr := this.Get()
		if caught == thisErr {
			for _, handler := range handlers {
				handler(thisErr)
			}
			return true
		}
		return false
	}
}
