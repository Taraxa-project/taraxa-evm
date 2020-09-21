package goroutines

import (
	"sync"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type SequentialTaskGroupExecutor struct {
	pending_groups chan *SequentialTaskGroup
}

func (self *SequentialTaskGroupExecutor) Init(queue_size uint32, num_goroutines int) *SequentialTaskGroupExecutor {
	self.pending_groups = make(chan *SequentialTaskGroup, queue_size)
	for i := 0; i < num_goroutines; i++ {
		go func() {
			for {
				g, still_open := <-self.pending_groups
				if !still_open {
					return
				}
				g.execute_pending()
			}
		}()
	}
	return self
}

// TODO use
func (self *SequentialTaskGroupExecutor) Close() {
	// not graceful
	close(self.pending_groups)
}

type SequentialTaskGroup struct {
	host_queue         chan *SequentialTaskGroup
	lock               sync.Mutex
	pending_tasks      []func()
	tasks_in_execution []func()
}

func (self *SequentialTaskGroupExecutor) NewGroup(initial_buffer_size uint32) SequentialTaskGroup {
	return SequentialTaskGroup{
		host_queue:         self.pending_groups,
		pending_tasks:      make([]func(), 0, initial_buffer_size),
		tasks_in_execution: make([]func(), 0, initial_buffer_size),
	}
}

func (self *SequentialTaskGroup) Submit(task func()) {
	self.lock.Lock()
	was_empty := len(self.pending_tasks) == 0 && len(self.tasks_in_execution) == 0
	self.pending_tasks = append(self.pending_tasks, task)
	self.lock.Unlock()
	if was_empty {
		self.host_queue <- self
	}
}

func (self *SequentialTaskGroup) Join() {
	self.lock.Lock()
	if len(self.pending_tasks) == 0 && len(self.tasks_in_execution) == 0 {
		self.lock.Unlock()
		return
	}
	var m sync.Mutex
	m.Lock()
	self.pending_tasks = append(self.pending_tasks, m.Unlock)
	self.lock.Unlock()
	m.Lock()
}

func (self *SequentialTaskGroup) execute_pending() {
	for {
		self.lock.Lock()
		if len(self.pending_tasks) == 0 {
			bin.ZFill_3(unsafe.Pointer(&self.tasks_in_execution), unsafe.Sizeof(self.tasks_in_execution[0]))
			self.tasks_in_execution = self.tasks_in_execution[:0]
			self.lock.Unlock()
			return
		}
		self.tasks_in_execution, self.pending_tasks = self.pending_tasks, self.tasks_in_execution
		bin.ZFill_3(unsafe.Pointer(&self.pending_tasks), unsafe.Sizeof(self.pending_tasks[0]))
		self.pending_tasks = self.pending_tasks[:0]
		self.lock.Unlock()
		for _, task := range self.tasks_in_execution {
			task()
		}
	}
}
