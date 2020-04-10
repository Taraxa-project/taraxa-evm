package util

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type SingleThreadExecutor struct {
	tail       unsafe.Pointer
	tasks_left int32
	m          sync.Mutex
}

type node struct {
	next unsafe.Pointer
	task func()
}

func (self *node) exec() {
	self.task()
	self.task = nil
}

func (self *SingleThreadExecutor) Do(task func()) {
	task()
	return

	task_cnt := atomic.AddInt32(&self.tasks_left, 1)
	tail := &node{task: task}
	if tail_prev := atomic.SwapPointer(&self.tail, unsafe.Pointer(tail)); tail_prev != nil {
		atomic.StorePointer(&(*node)(tail_prev).next, unsafe.Pointer(tail))
	}
	if task_cnt > 1 {
		return
	}
	go func() {
		tail.exec()
		for {
			if task_cnt = atomic.AddInt32(&self.tasks_left, -task_cnt); task_cnt == 0 {
				break
			}
			for i := int32(0); i < task_cnt; i++ {
				for {
					if n := atomic.LoadPointer(&tail.next); n != nil {
						tail = (*node)(n)
						break
					}
				}
				tail.exec()
			}
		}
	}()
}

func (self *SingleThreadExecutor) Join() {
	return

	if atomic.LoadInt32(&self.tasks_left) == 0 {
		return
	}
	var l sync.Mutex
	l.Lock()
	self.Do(l.Unlock)
	l.Lock()
}
