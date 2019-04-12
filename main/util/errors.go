package util

import (
	"errors"
	"strings"
)

type Hanlder func(interface{}) bool
type ErrorHandler func(error)

func SetTo(errPtr *error) ErrorHandler {
	return func(err error) {
		*errPtr = err
	}
}

func CatchAny(handlers ...func(interface{})) Hanlder {
	return func(caught interface{}) bool {
		for _, handler := range handlers {
			handler(caught)
		}
		return true
	}
}

func CatchAnyErr(handlers ...ErrorHandler) Hanlder {
	return func(caught interface{}) bool {
		if err, isErr := caught.(error); isErr {
			for _, handler := range handlers {
				handler(err)
			}
			return true
		}
		return false
	}
}

func Recover(handlers ...Hanlder) (caught interface{}) {
	if caught = recover(); caught != nil {
		if !Handle(caught, handlers...) {
			panic(caught)
		}
	}
	return
}

func Handle(obj interface{}, handlers ...Hanlder) bool {
	if len(handlers) == 0 {
		return true
	}
	for _, handler := range handlers {
		if handler(obj) {
			return true
		}
	}
	return false
}

func PanicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func Assert(condition bool, msg ...string) {
	if !condition {
		panic(errors.New(strings.Join(msg, "")))
	}
}

func Try(action func()) interface{} {
	return func() (recovered interface{}) {
		defer func() {
			recovered = recover()
		}()
		action()
		return
	}()
}
