package util

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"hash/crc64"
	"math/big"
	"reflect"
)

type Predicate func(interface{}) bool

func Sum(x, y *big.Int) *big.Int {
	if x == nil {
		x = common.Big0
	}
	if y == nil {
		y = common.Big0
	}
	return new(big.Int).Add(x, y)
}

func DoNothing() {}

func Chain(f, g func()) func() {
	return func() {
		f()
		g()
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

func isReallyNil(value interface{}) bool {
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

var CRC64_ISO_TABLE = crc64.MakeTable(crc64.ISO)

func CRC64(b []byte) uint64 {
	return crc64.Checksum(b, CRC64_ISO_TABLE)
}

func Either(cond bool, left, right interface{}) interface{} {
	if cond {
		return left
	}
	return right
}
