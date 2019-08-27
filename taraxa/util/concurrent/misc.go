package concurrent

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"reflect"
	"sync"
	"sync/atomic"
)

func TryClose(channel interface{}) error {
	ret := util.Try(func() {
		reflect.ValueOf(channel).Close()
	})
	if ret != nil {
		return ret.(error)
	}
	return nil
}

func TrySend(channel, value interface{}) error {
	ret := util.Try(func() {
		reflect.ValueOf(channel).Send(reflect.ValueOf(value))
	})
	if ret != nil {
		return ret.(error)
	}
	return nil
}

func WithLock(lock sync.Locker, action func()) {
	lock.Lock()
	defer lock.Unlock()
	action()
}

func LockUnlock(lock sync.Locker) (unlock func()) {
	lock.Lock()
	return lock.Unlock
}

func Parallelize(numGoroutines, N int, prepare func(goroutineIndex int) (action func(i int))) {
	util.Assert(N >= 0 && numGoroutines >= 0)
	if numGoroutines > N {
		numGoroutines = N
	}
	allDone := NewRendezvous(N)
	scheduledCount := uint32(0)
	for goroutineIndex := 0; goroutineIndex < numGoroutines; goroutineIndex++ {
		go func() {
			action := prepare(goroutineIndex)
			for {
				i := int(atomic.AddUint32(&scheduledCount, 1)) - 1
				if i >= N {
					break
				}
				func() {
					defer allDone.CheckIn()
					action(i)
				}()
			}
		}()
	}
	allDone.Await()
}
