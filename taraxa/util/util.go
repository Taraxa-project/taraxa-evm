package util

import (
	"math"
	"reflect"
	"sync"
)

type Any interface{}

func IsReallyNil(value Any) bool {
	if value == nil {
		return true
	}
	switch reflectValue := reflect.ValueOf(value); reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return reflectValue.IsNil()
	default:
		return false
	}
}

func MaxU64(x, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
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

func PanicIfNotNil(value interface{}) bool {
	if !IsReallyNil(value) {
		panic(value)
	}
	return true
}

func CeilPow2(x int) int {
	return 1 << uint(math.Ceil(math.Log2(float64(x))))
}

func Recover(handler func(issue Any)) {
	if r := recover(); r != nil {
		handler(r)
	}
}

func Min(i, j int) int {
	if i < j {
		return i
	}
	return j
}

func MinU64(i, j uint64) uint64 {
	if i < j {
		return i
	}
	return j
}

func LockUnlock(l sync.Locker) func() {
	l.Lock()
	return l.Unlock
}

// a nice way to create a block of code with scope
func Call(f func()) {
	f()
}
