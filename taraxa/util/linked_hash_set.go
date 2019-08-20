package util

import "github.com/emirpasic/gods/sets/linkedhashset"

type LinkedHashSet struct {
	*linkedhashset.Set
}

func NewLinkedHashSet(values interface{}) (this *LinkedHashSet) {
	this = &LinkedHashSet{linkedhashset.New()}
	if values == nil {
		return
	}
	ForEach(values, func(i int, val interface{}) {
		this.Add(val)
	})
	return
}

func (this *LinkedHashSet) UnmarshalJSON(b []byte) error {
	this.Set = linkedhashset.New()
	return this.FromJSON(b)
}

func (this *LinkedHashSet) MarshalJSON() ([]byte, error) {
	return this.ToJSON()
}

func (this *LinkedHashSet) String() string {
	return Join(", ", this.Values())
}
