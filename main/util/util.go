package util

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"math/big"
	"reflect"
	"sync/atomic"
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

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

type Interval struct {
	StartInclusive, EndExclusive int
}

func (this *Interval) IsEmpty() bool {
	return this.StartInclusive == this.EndExclusive
}

type AtomicRange struct {
	left int32
}

func NewAtomicRange(size int) AtomicRange {
	return AtomicRange{int32(size)}
}

func (this *AtomicRange) Take(size int) *Interval {
	newVal := int(atomic.AddInt32(&this.left, -int32(size)));
	return &Interval{Max(newVal, 0), Max(newVal+size, 0)}
}

func (this *AtomicRange) IsEmpty() bool {
	return atomic.LoadInt32(&this.left) == 0
}

type IncreasingAtomicRange struct {
	taken int32
	size  int
}

func NewIncreasingAtomicRange(size int) IncreasingAtomicRange {
	return IncreasingAtomicRange{size: size}
}

func (this *IncreasingAtomicRange) Take(size int) *Interval {
	newVal := int(atomic.AddInt32(&this.taken, int32(size)));
	return &Interval{Min(newVal-size, this.size), Min(newVal, this.size)}
}

func (this *IncreasingAtomicRange) IsEmpty() bool {
	return atomic.LoadInt32(&this.taken) == int32(this.size)
}
