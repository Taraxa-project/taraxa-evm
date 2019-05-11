package util

//import "reflect"
//
//func CloseChan(channel interface{}) error {
//	ret := Try(func() {
//		reflect.ValueOf(channel).Close()
//	})
//	if ret != nil {
//		return ret.(error)
//	}
//	return nil
//}
//
//type ConcurrentGroup chan interface{}
//
//func (this *ConcurrentGroup) Submit(task func(*ConcurrentGroup) interface{}) *ConcurrentGroup {
//	go task(this)
//}
