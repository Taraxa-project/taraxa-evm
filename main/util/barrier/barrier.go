package barrier

import (
	"sync/atomic"
)

type Barrier struct {
	count int32
}

func New(capacity int) Barrier {
	return Barrier{
		count: int32(capacity),
	}
}

func (this *Barrier) CheckIn() {
	atomic.AddInt32(&this.count, -1)
}

func (this *Barrier) Await() {
	for atomic.LoadInt32(&this.count) > 0 {

	}
}
