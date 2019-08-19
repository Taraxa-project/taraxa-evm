package util

import (
	"reflect"
)

func TryClose(channel interface{}) error {
	ret := Try(func() {
		reflect.ValueOf(channel).Close()
	})
	if ret != nil {
		return ret.(error)
	}
	return nil
}

func TrySend(channel, value interface{}) error {
	ret := Try(func() {
		reflect.ValueOf(channel).Send(reflect.ValueOf(value))
	})
	if ret != nil {
		return ret.(error)
	}
	return nil
}

//
//type ConcurrentGroup chan interface{}
//
//func (this *ConcurrentGroup) Submit(task func(*ConcurrentGroup) interface{}) *ConcurrentGroup {
//	go task(this)
//}

type SimpleThreadPool struct {
	taskQueue chan func()
}

func LaunchNewSimpleThreadPool(threadCount, queueSize int) *SimpleThreadPool {
	this := &SimpleThreadPool{make(chan func(), queueSize)}
	for i := 0; i < threadCount; i++ {
		go func() {
			for {
				task, ok := <-this.taskQueue
				if !ok {
					break
				}
				task()
			}
		}()
	}
	return this
}

func (this *SimpleThreadPool) Go(task func()) error {
	return TrySend(this.taskQueue, task)
}

func (this *SimpleThreadPool) Close() error {
	return TryClose(this.taskQueue)
}
