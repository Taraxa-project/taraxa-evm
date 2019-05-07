package util

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"math/big"
	"reflect"
)

func Sum(x, y *big.Int) *big.Int {
	if x == nil {
		x = common.Big0
	}
	if y == nil {
		y = common.Big0
	}
	return new(big.Int).Add(x, y)
}

func DoNothing() {

}

func Noop(...interface{}) interface{} {
	return nil
}

func Chain(f, g func()) func() {
	return func() {
		f()
		g()
	}
}

type Remapper func(key, oldValue interface{}, wasPresent bool) (newValue interface{})

func Compute(anyMap, key interface{}, remapper Remapper) (newValue interface{}, revert func()) {
	reflectMap := reflect.ValueOf(anyMap)
	reflectKey := reflect.ValueOf(key)
	reflectOldVal := reflectMap.MapIndex(reflectKey)
	wasPresent := reflectOldVal.IsValid()
	var oldVal interface{}
	if wasPresent {
		oldVal = reflectOldVal.Interface()
	} else {
		valueType := reflectMap.Type().Elem()
		oldVal = reflect.Zero(valueType).Interface()
	}
	newValue = remapper(key, oldVal, wasPresent)
	reflectMap.SetMapIndex(reflectKey, reflect.ValueOf(newValue))
	return newValue, func() {
		reflectMap.SetMapIndex(reflectKey, reflectOldVal)
	}
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
