package util

import "github.com/emirpasic/gods/sets/linkedhashset"

type LinkedHashSet struct {
	*linkedhashset.Set
}

func NewLinkedHashSet(values interface{}) *LinkedHashSet {
	ret := &LinkedHashSet{linkedhashset.New()}
	if values != nil {
		ForEach(values, func(i int, val interface{}) {
			ret.Add(val)
		})
	}
	return ret
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
