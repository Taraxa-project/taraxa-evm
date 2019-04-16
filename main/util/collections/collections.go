package collections

import (
	"github.com/emirpasic/gods/sets/linkedhashset"
)

func Len(set *linkedhashset.Set) int {
	if set == nil {
		return 0
	}
	return set.Size()
}
