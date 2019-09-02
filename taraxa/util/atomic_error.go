package util

import (
	"sync/atomic"
)

type AtomicError struct {
	present int32
	err     atomic.Value
}

func (this *AtomicError) SetIfAbsent(err error) (hasSet bool) {
	if isReallyNil(err) || !atomic.CompareAndSwapInt32(&this.present, 0, 1) {
		return false
	}
	this.err.Store(err)
	return true
}

func (this *AtomicError) SetAndCheck(err error) bool {
	return this.SetIfAbsent(err) || this.Check()
}

func (this *AtomicError) Check() bool {
	return atomic.CompareAndSwapInt32(&this.present, 1, 1)
}

func (this *AtomicError) SetOrPanicIfPresent(errors ...error) {
	this.PanicIfPresent()
	for _, err := range errors {
		if this.SetIfAbsent(err) {
			panic(err)
		}
	}
}

func (this *AtomicError) PanicIfPresent() {
	PanicIfNotNil(this.Get())
}

func (this *AtomicError) Get() error {
	val := this.err.Load();
	if val == nil {
		return nil
	}
	return val.(error)
}

func (this *AtomicError) Catch(handlers ...ErrorHandler) Predicate {
	return func(caught interface{}) bool {
		if thisErr := this.Get(); caught == thisErr {
			for _, handler := range handlers {
				handler(thisErr)
			}
			return true
		}
		return false
	}
}

func (this *AtomicError) Recover(target *interface{}) {
	*target = Recover(this.Catch())
}
