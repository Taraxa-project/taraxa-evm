package util

import (
	"reflect"
	"sync"
)

type Pool struct {
	factory         func() PoolItem
	handles         chan PoolItemHandle
	handles_reflect reflect.Value
	generation      uint32
	invalidate_mu   sync.RWMutex
}
type PoolItem interface {
	Close()
}
type PoolItemHandle struct {
	item       PoolItem
	generation uint32
}

func (self *PoolItemHandle) Get() interface{} {
	return self.item
}

func (self *Pool) Init(capacity uint, factory func() PoolItem) *Pool {
	self.factory = factory
	self.reset_handles(capacity)
	return self
}

func (self *Pool) reset_handles(size uint) {
	self.handles = make(chan PoolItemHandle, size)
	self.handles_reflect = reflect.ValueOf(self.handles)
}

func (self *Pool) make_handle() (ret PoolItemHandle) {
	ret.item = self.factory()
	ret.generation = self.generation
	return
}

func (self *Pool) Get() (ret PoolItemHandle) {
	defer LockUnlock(self.invalidate_mu.RLocker())()
	select {
	case ret = <-self.handles:
	default:
		ret = self.make_handle()
	}
	return
}

func (self *Pool) Return(obj PoolItemHandle) {
	defer LockUnlock(self.invalidate_mu.RLocker())()
	if obj.generation != self.generation || !self.handles_reflect.TrySend(reflect.ValueOf(obj)) {
		obj.item.Close()
	}
}

func (self *Pool) Invalidate() (cleanup func()) {
	defer LockUnlock(&self.invalidate_mu)()
	self.generation++
	handles_old := self.handles
	self.reset_handles(uint(cap(handles_old)))
	return func() {
		for left := len(handles_old); left != 0; left-- {
			(<-handles_old).item.Close()
		}
	}
}
