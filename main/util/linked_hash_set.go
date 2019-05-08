package util

import "github.com/emirpasic/gods/sets/linkedhashset"

type LinkedHashSet struct {
	*linkedhashset.Set
}

func (this *LinkedHashSet) UnmarshalJSON(b []byte) error {
	this.Set = linkedhashset.New()
	return this.FromJSON(b)
}

func (this *LinkedHashSet) MarshalJSON() ([]byte, error) {
	return this.ToJSON()
}
