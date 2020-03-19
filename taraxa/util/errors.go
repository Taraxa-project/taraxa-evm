package util

import (
	"errors"
	"strings"
)

type ErrorHandler func(_ error)

type ErrorString string

func (this ErrorString) Error() string {
	return string(this)
}

func Stringify(err_ptr *error) {
	if err := *err_ptr; err != nil {
		*err_ptr = ErrorString(err.Error())
	}
}

func SetTo(errPtr *error) ErrorHandler {
	return func(err error) {
		*errPtr = err
	}
}

func CatchAnyErr(handlers ...ErrorHandler) Predicate {
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

func Recover(errorFilters ...Predicate) (caught interface{}) {
	if caught = recover(); caught != nil {
		if !AnyMatches(caught, errorFilters...) {
			panic(caught)
		}
	}
	return
}

func AnyMatches(obj interface{}, handlers ...Predicate) bool {
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

func PanicIfNotNil(value interface{}) {
	if !IsReallyNil(value) {
		panic(value)
	}
}

func Assert(condition bool, msg ...string) bool {
	if !condition {
		panic(errors.New(strings.Join(msg, " ")))
	}
	return true
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
