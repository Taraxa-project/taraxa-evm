package util

import (
	"errors"
	"reflect"
	"strings"
)

type Any = interface{}

func IsReallyNil(value interface{}) bool {
	if value == nil {
		return true
	}
	switch reflectValue := reflect.ValueOf(value); reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return reflectValue.IsNil()
	}
	return false
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

type ErrorString string

func (this ErrorString) Error() string {
	return string(this)
}

func Stringify(err_ptr *error) {
	if err := *err_ptr; err != nil {
		*err_ptr = ErrorString(err.Error())
	}
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

func Recover(handler func(issue Any)) {
	if r := recover(); r != nil {
		handler(r)
	}
}
