package metric_utils

import (
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
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

func (this *AtomicCounter) NewTimeRecorder() func() time.Duration {
	return NewTimeRecorder(this.AddDuration)
}

func (this *AtomicCounter) MeasureElapsedTime(action func()) *AtomicCounter {
	recorder := this.NewTimeRecorder()
	defer recorder()
	action()
	return this
}

func MeasureElapsedTime(counters ...*AtomicCounter) proxy.Decorator {
	return func(...proxy.Argument) proxy.ArgumentsCallback {
		recordTime := NewTimeRecorder(func(duration time.Duration) {
			for _, counter := range counters {
				counter.AddDuration(duration)
			}
		})
		return func(...proxy.Argument) {
			recordTime()
		}
	}
}
