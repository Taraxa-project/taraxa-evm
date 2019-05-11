package proxy

import "sync"

type Argument = interface{}
type ArgumentsCallback func(arguments ...Argument)
type Decorator func(arguments ...Argument) ArgumentsCallback
type DecoratorMap = map[string]Decorator

type Decorators struct {
	m sync.Map
}

func (this *Decorators) Register(name string, decorator Decorator) *Decorators {
	this.m.Store(name, decorator)
	return this
}

func (this *Decorators) RegisterMany(decoratorMap DecoratorMap) *Decorators {
	for k, v := range decoratorMap {
		this.Register(k, v)
	}
	return this
}

func (this *Decorators) BeforeCall(name string, args ...Argument) (afterCall ArgumentsCallback) {
	if beforeCall, ok := this.m.Load(name); ok {
		if afterCall = beforeCall.(Decorator)(args...); afterCall != nil {
			return afterCall
		}
	}
	return func(argument ...Argument) {}
}

func (this *Decorators) Delete(names ...string) *Decorators {
	for name := range names {
		this.m.Delete(name)
	}
	return this
}
