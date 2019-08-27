package metric_utils

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/proxy"
	"sync/atomic"
	"time"
)

type AtomicCounter uint64

func (this *AtomicCounter) Add(x uint64) {
	atomic.AddUint64((*uint64)(this), x)
}

func (this *AtomicCounter) AddDuration(d time.Duration) {
	this.Add(uint64(d.Nanoseconds()))
}

func (this *AtomicCounter) Get() uint64 {
	return atomic.LoadUint64((*uint64)(this))
}

func (this *AtomicCounter) Recorder() func() {
	start := time.Now()
	return func() {
		this.AddDuration(time.Since(start))
	}
}

func (this *AtomicCounter) RecordElapsedTime(action func()) *AtomicCounter {
	recorder := this.Recorder()
	defer recorder()
	action()
	return this
}

func (this *AtomicCounter) Decorator() proxy.Decorator {
	return func(...proxy.Argument) func(...proxy.Argument) {
		recordTime := this.Recorder()
		return func(...proxy.Argument) {
			recordTime()
		}
	}
}
