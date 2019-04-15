package collections

import (
	"github.com/emirpasic/gods/sets/linkedhashset"
)

func ContainsExactly(set *linkedhashset.Set, elements ...interface{}) bool {
	return set.Size() == len(elements) && set.Contains(elements...)
}

func IsEmpty(set *linkedhashset.Set) bool {
	return set == nil || set.Empty()
}