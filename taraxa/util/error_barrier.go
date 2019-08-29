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

func (this *ErrorBarrier) SetAndCheck(err error) bool {
	return this.SetIfAbsent(err) || this.Check()
}

func (this *ErrorBarrier) Check() bool {
	return atomic.CompareAndSwapInt32(&this.active, 1, 1)
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
	PanicIfNotNil(this.Get())
}

func (this *ErrorBarrier) Get() error {
	val := this.err.Load();
	if val == nil {
		return nil
	}
	return val.(error)
}

func (this *ErrorBarrier) Catch(handlers ...ErrorHandler) Predicate {
	return func(caught interface{}) bool {
		if thisErr := this.Get(); thisErr != nil && caught == thisErr {
			for _, handler := range handlers {
				handler(thisErr)
			}
			return true
		}
		return false
	}
}
