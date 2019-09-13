package main

import (
	"fmt"
	"unsafe"
)

type Foo struct {
	i int
}

func (this *Foo) bar() {
	fmt.Println((unsafe.Pointer)(this))
}

func NewFoo() (ret Foo) {
	ret.bar()
	return
}

func main() {
	f := NewFoo()
	f.bar()
}
