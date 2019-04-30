package barrier

import (
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"sync/atomic"
)

type Barrier struct {
	checkInsLeft int32
	// Chan is used to make sure we wait using gopark
	waitChan     chan interface{}
}

func New(size int) *Barrier {
	util.Assert(size > 0, "size must be > 0")
	ret := new(Barrier)
	ret.checkInsLeft = int32(size)
	ret.waitChan = make(chan interface{})
	return ret
}

func (this *Barrier) CheckIn() {
	if atomic.AddInt32(&this.checkInsLeft, -1) == 0 {
		close(this.waitChan)
	}
}

func (this *Barrier) Await() {
	<-this.waitChan
}
