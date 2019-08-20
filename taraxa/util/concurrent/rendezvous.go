package concurrent

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"sync/atomic"
)

type Rendezvous struct {
	checkInsLeft int64
	// Chan is used to make sure we wait using gopark
	waitChan chan interface{}
}

func NewRendezvous(size int) *Rendezvous {
	util.Assert(size >= 0, "size must be >= 0")
	this := new(Rendezvous)
	this.checkInsLeft = int64(size)
	this.waitChan = make(chan interface{})
	if this.checkInsLeft == 0 {
		close(this.waitChan)
	}
	return this
}

func (this *Rendezvous) CheckIn() (left int64) {
	if left = atomic.AddInt64(&this.checkInsLeft, -1); left == 0 {
		close(this.waitChan)
	}
	return
}

func (this *Rendezvous) Await() {
	<-this.waitChan
}
