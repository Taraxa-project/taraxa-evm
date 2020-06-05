package util

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type SingleThreadExecutor struct {
	tail unsafe.Pointer
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
	n := &node{task: task}
	if tail_prev := atomic.SwapPointer(&self.tail, unsafe.Pointer(n)); tail_prev != nil {
		atomic.StorePointer(&(*node)(tail_prev).next, unsafe.Pointer(n))
		return
	}
	go func() {
		n.exec()
		tail := unsafe.Pointer(n)
		for {
			if tail == unsafe.Pointer(n) {
				if atomic.CompareAndSwapPointer(&self.tail, tail, nil) {
					return
				}
				tail = atomic.LoadPointer(&self.tail)
			}
			if next := atomic.LoadPointer(&n.next); next != nil {
				n = (*node)(next)
				n.exec()
			}
		}
	}()
}

func (self *SingleThreadExecutor) Synchronize() {
	if atomic.LoadPointer(&self.tail) == nil {
		return
	}
	l := new(sync.Mutex)
	l.Lock()
	self.Do(l.Unlock)
	l.Lock()
}
