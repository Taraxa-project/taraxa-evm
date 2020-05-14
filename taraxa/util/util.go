package util

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"math"
	"math/rand"
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

func RandomBytes(desired_len int, rnd *rand.Rand) []byte {
	ret := make([]byte, 0, desired_len)
	for {
		curr_len := len(ret)
		if curr_len == desired_len {
			break
		}
		bin.ENC_b_endian_compact_64(rnd.Uint64(), func(b ...byte) {
			ret = append(ret, b[:Min(len(b), desired_len-curr_len)]...)
		})
	}
	return ret
}

func LockUnlock(l sync.Locker) func() {
	l.Lock()
	return l.Unlock
}
