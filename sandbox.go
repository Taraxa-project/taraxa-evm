package main

import (
	"fmt"
	"reflect"
)

func main() {
	var b *int
	var foo interface{} = b
	fmt.Println(reflect.ValueOf(foo).IsNil())
}
