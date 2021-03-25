package goroutines

import (
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
)

type GoroutineGroup struct {
	tasks       chan func()
	num_running int64
}

func (self *GoroutineGroup) Init(goroutine_count uint32, buffer_size uint32) *GoroutineGroup {
	self.tasks = make(chan func(), buffer_size)
	self.num_running = int64(goroutine_count)
	for i := uint32(0); i < goroutine_count; i++ {
		go func() {
			defer atomic.AddInt64(&self.num_running, -1)
			for {
				if task, ok := <-self.tasks; ok {
					task()
				} else {
					break
				}
			}
		}()
	}
	return self
}

func (self *GoroutineGroup) InitSingle(buffer_size uint32) *GoroutineGroup {
	return self.Init(1, buffer_size)
}

func (self *GoroutineGroup) Submit(task func()) {
	self.tasks <- task
}

func (self *GoroutineGroup) Join() {
	var m sync.Mutex
	m.Lock()
	self.Submit(m.Unlock)
	m.Lock()
}

func (self *GoroutineGroup) JoinAndClose() {
	self.Submit(func() {
		close(self.tasks)
	})
	for atomic.LoadInt64(&self.num_running) != 0 {
		runtime.Gosched()
	}
	asserts.Holds(len(self.tasks) == 0)
}
