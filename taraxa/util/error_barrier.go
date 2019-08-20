package util

import (
	"sync/atomic"
)

type ErrorBarrier struct {
	active int32
	err    atomic.Value
}

func (this *ErrorBarrier) SetIfAbsent(err error) (hasSet bool) {
	if isReallyNil(err) || !atomic.CompareAndSwapInt32(&this.active, 0, 1) {
		return false
	}
	this.err.Store(err)
	return true
}

func (this *ErrorBarrier) CheckIn(errors ...error) {
	this.PanicIfPresent()
	for _, err := range errors {
		if this.SetIfAbsent(err) {
			panic(err)
		}
	}
}

func (this *ErrorBarrier) PanicIfPresent() {
	PanicIfPresent(this.Get())
}

func (this *ErrorBarrier) Get() error {
	return this.err.Load().(error)
}

func (this *ErrorBarrier) Catch(handlers ...ErrorHandler) Predicate {
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
