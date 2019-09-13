package concurrent

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

var CPU_COUNT = runtime.NumCPU()

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

type IterationController = func(iteration int) (shouldExit bool)

func Parallelize(numGoroutines, N int, prepare func(goroutineIndex int) IterationController) {
	util.Assert(N >= 0 && numGoroutines >= 0)
	if numGoroutines > N {
		numGoroutines = N
	}
	allDone := NewRendezvous(numGoroutines)
	for goroutineIndex, scheduledCount := 0, uint32(0); goroutineIndex < numGoroutines; goroutineIndex++ {
		goroutineIndex := goroutineIndex
		go func() {
			defer allDone.CheckIn()
			for action := prepare(goroutineIndex); ; {
				if i := int(atomic.AddUint32(&scheduledCount, 1)) - 1; i < N {
					if shouldExit := action(i); !shouldExit {
						continue
					}
				}
				return
			}
		}()
	}
	allDone.Await()
}

func SimpleParallelize(numGoroutines, N int, action func(i int)) {
	Parallelize(numGoroutines, N, func(int) IterationController {
		return func(i int) bool {
			action(i)
			return false
		}
	})
}
