package concurrent

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"sync"
	"sync/atomic"
)

type AtomicError struct {
	present int32
	// TODO pointer to error
	error          atomic.Value
	callbacks      []util.ErrorHandler
	callbacksMutex sync.Mutex
}

func (this *AtomicError) AddHandler(handler util.ErrorHandler) {
	defer LockUnlock(&this.callbacksMutex)()
	this.callbacks = append(this.callbacks, handler)
}

func (this *AtomicError) SetIfAbsent(err error) (hasSet bool) {
	if util.IsReallyNil(err) || !atomic.CompareAndSwapInt32(&this.present, 0, 1) {
		return false
	}
	// TODO this is a race
	this.error.Store(err)
	defer LockUnlock(&this.callbacksMutex)()
	for _, callback := range this.callbacks {
		callback(err)
	}
	return true
}

func (this *AtomicError) SetAndCheck(err error) bool {
	return this.SetIfAbsent(err) || this.IsPresent()
}

func (this *AtomicError) IsPresent() bool {
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
	util.PanicIfNotNil(this.Get())
}

func (this *AtomicError) Get() error {
	val := this.error.Load();
	if val == nil {
		return nil
	}
	return val.(error)
}

func (this *AtomicError) Catch(handlers ...util.ErrorHandler) util.Predicate {
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

func (this *AtomicError) Recover() error {
	return util.Recover(this.Catch()).(error)
}

func (this *AtomicError) RecoverAndSetTo(target *error) {
	*target = this.Recover()
}
