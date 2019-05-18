package barrier

import (
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"sync/atomic"
)

type Barrier struct {
	checkInsLeft int32
	// Chan is used to make sure we wait using gopark
	waitChan chan interface{}
}

func New(size int) *Barrier {
	util.Assert(size >= 0, "size must be >= 0")
	this := new(Barrier)
	this.checkInsLeft = int32(size)
	this.waitChan = make(chan interface{})
	if this.checkInsLeft == 0 {
		close(this.waitChan)
	}
	return this
}

func (this *Barrier) CheckIn() (left int32) {
	if left = atomic.AddInt32(&this.checkInsLeft, -1); left == 0 {
		close(this.waitChan)
	}
	return
}

func (this *Barrier) Await() {
	<-this.waitChan
}
