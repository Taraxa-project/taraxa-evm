package main

import "fmt"

type O func()


type foo struct {
	i int
	s string
}

type Foo func()

func (this Foo) call()  {
	fmt.Println("foo")
}

func main() {
	var foo map[string]int
	fmt.Println(foo[""])
}
