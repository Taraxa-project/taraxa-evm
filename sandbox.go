package main

import "fmt"

type foo struct {
	i int
	j string
}

func main() {
	//var f = new(foo)
	var foo = []int{1, 2, 3}
	fmt.Println(foo[:3])
}
