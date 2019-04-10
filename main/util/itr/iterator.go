package itr

import (
	"github.com/emirpasic/gods/sets/treeset"
	"reflect"
)

type cmd uint64

const END = cmd(0)

var endType reflect.Type = reflect.TypeOf(END)

func IsEnd(val interface{}) bool {
	return val == END && reflect.TypeOf(val) == endType
}

type Iterator func() interface{}
type IntIterator func() (int, bool)

func (this Iterator) Int() IntIterator {
	return func() (ret int, done bool) {
		if v := this(); IsEnd(v) {
			done = true
		} else {
			ret = v.(int)
		}
		return
	}
}

func From(value ...interface{}) Iterator {
	i := 0
	return func() interface{} {
		if i < len(value) {
			i++
			return value[i-1]
		}
		return END
	}
}

func FromTreeSet(set *treeset.Set) Iterator {
	itr := set.Iterator()
	return func() interface{} {
		if (itr.Next()) {
			return itr.Value()
		}
		return END
	}
}
