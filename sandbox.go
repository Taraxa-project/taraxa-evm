package main

import "fmt"

func main() {
	var i interface{}
	i, ok := i.(*int)
	fmt.Println(i, ok)
}
