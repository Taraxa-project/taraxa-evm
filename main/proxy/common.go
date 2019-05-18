package proxy

import (
	"sync"
	"sync/atomic"
)

type Argument = interface{}
type Decorator func(arguments ...Argument) func(returnArgs ...Argument)

type Proxy interface {
	RegisterDecorator(name string, decorator Decorator) (unregister func())
}

type BaseProxy struct {
	storage         sync.Map
	lastDecoratorId uint64
}

func (this *BaseProxy) RegisterDecorator(name string, decorator Decorator) (unregister func()) {
	v, _ := this.storage.LoadOrStore(name, new(sync.Map))
	idToDecoratorMap := v.(*sync.Map)
	id := atomic.AddUint64(&this.lastDecoratorId, 1)
	idToDecoratorMap.Store(id, decorator)
	return func() {
		idToDecoratorMap.Delete(id)
	}
}

func (this *BaseProxy) CallDecorator(name string, args ...Argument) (afterCall func(...Argument)) {
	afterCall = func(argument ...Argument) {}
	if v, present := this.storage.Load(name); present {
		v.(*sync.Map).Range(func(_, value interface{}) bool {
			callback := value.(Decorator)(args...)
			prevCallbacks := afterCall
			afterCall = func(arguments ...Argument) {
				callback(arguments...)
				prevCallbacks(arguments...)
			}
			return true
		})
	}
	return
}

func TryRegisterDecorator(obj interface{}, name string, decorator Decorator) (unregister func()) {
	if proxy, isProxy := obj.(Proxy); isProxy {
		return proxy.RegisterDecorator(name, decorator)
	}
	return func() {}
}
