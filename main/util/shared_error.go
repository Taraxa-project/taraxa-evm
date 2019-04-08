package util

//import "sync/atomic"

type SharedError struct {
}

//func (this *SharedError) GetAndSet(err error) error {
//	atomic.CompareAndSwapPointer()
//
//}

func (this *SharedError) SetIfAbsent(err error) (hasSet bool) {

}

func (this *SharedError) SetIfAbsentAndPanic(err error) (hasSet bool) {

}

func (this *SharedError) CheckIn(errors... error) {
	this.PanicIfPresent()
	for _, err := range errors {
		this.SetIfAbsent(err)
		this.PanicIfPresent()
	}
}

func (this *SharedError) Get() error {

}

func (this *SharedError) PanicIfPresent() {

}

func (this *SharedError) Recover(cb ...func(error)) {

}