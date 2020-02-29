package util

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
)

type Predicate func(interface{}) bool

func DoNothing() {}

func Chain(f, g func()) func() {
	return func() {
		defer g()
		f()
	}
}

type Remapper func(key, oldValue interface{}, wasPresent bool) (newValue interface{})

func ForEach(indexableWithLength interface{}, cb func(i int, val interface{})) {
	val := reflect.ValueOf(indexableWithLength)
	length := val.Len()
	for i := 0; i < length; i++ {
		cb(i, val.Index(i).Interface())
	}
}

func Join(separator string, indexableWithLength interface{}) (result string) {
	length := reflect.ValueOf(indexableWithLength).Len()
	ForEach(indexableWithLength, func(i int, val interface{}) {
		result += fmt.Sprint(val)
		if i < length-1 {
			result += separator
		}
	})
	return
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func IsReallyNil(value interface{}) bool {
	if value == nil {
		return true
	}
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return reflectValue.IsNil()
	}
	return false
}

func RandomBytes(N int) (ret []byte) {
	buf := new(big.Int)
	for len(ret) < N {
		ret = append(ret, buf.SetUint64(rand.Uint64()).Bytes()...)
	}
	return ret[:N]
}

//func PooledFactory(new func() interface{}, reset func(interface{})) func() (ret interface{}, return_to_pool func()) {
//	pool := sync.Pool{new}
//	return func() (ret interface{}, return_to_pool func()) {
//		return pool.Get(), func() {
//			reset(ret)
//			pool.Put(ret)
//		}
//	}
//}
