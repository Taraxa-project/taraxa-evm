package proxy

type Argument = interface{}
type ArgumentsCallback func(arguments ...Argument)
type Decorator func(arguments ...Argument) ArgumentsCallback
type Decorators map[string]Decorator

func (this Decorators) BeforeCall(name string, args ...Argument) (afterCall ArgumentsCallback) {
	if beforeCall := this[name]; beforeCall != nil {
		return beforeCall(args...)
	}
	return func(argument ...Argument) {}
}
