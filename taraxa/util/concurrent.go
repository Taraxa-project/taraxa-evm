package util

import (
	"reflect"
	"sync"
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

func WithLock(lock sync.Locker, action func()) {
	lock.Lock()
	defer lock.Unlock()
	action()
}

func LockUnlock(lock sync.Locker) (unlock func()) {
	lock.Lock()
	return lock.Unlock
}
