package util

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
)

type Predicate func(interface{}) bool
type ErrorHandler func(error)

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

func PanicIfPresent(value interface{}) {
	if !IsNil(value) {
		fmt.Println(string(debug.Stack()))
		panic(value)
	}
}

func IsNil(value interface{}) bool {
	if value == nil {
		return true
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return reflectValue.IsNil()
	}
	return false
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
