package goroutines

import (
	"sync"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
)

type SingleThreadExecutor struct {
	tasks  chan func()
	closed bool
}

func (self *SingleThreadExecutor) Init(buffer_size uint32) *SingleThreadExecutor {
	self.tasks = make(chan func(), buffer_size)
	go func() {
		for !self.closed {
			(<-self.tasks)()
		}
		asserts.Holds(len(self.tasks) == 0)
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

func (self *SingleThreadExecutor) JoinAndClose() {
	var m sync.Mutex
	m.Lock()
	self.Submit(func() {
		self.closed = true
		m.Unlock()
	})
	m.Lock()
}
