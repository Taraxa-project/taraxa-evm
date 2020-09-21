package goroutines

import (
	"sync"
)

type SingleThreadExecutor struct {
	tasks chan func()
}

func (self *SingleThreadExecutor) Init(buffer_size uint32) *SingleThreadExecutor {
	self.tasks = make(chan func(), buffer_size)
	go func() {
		for {
			t, ok := <-self.tasks
			if !ok {
				return
			}
			t()
		}
	}()
	return self
}

func (self *SingleThreadExecutor) Submit(task func()) {
	self.tasks <- task
}

func (self *SingleThreadExecutor) Join() {
	var m sync.Mutex
	m.Lock()
	self.Submit(m.Unlock)
	m.Lock()
}

func (self *SingleThreadExecutor) Close() {
	// TODO
	panic("")
}
