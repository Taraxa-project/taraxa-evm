package main

import (
	"fmt"
)

type O func()

func main() {
	defer bar(foo())
	fmt.Println("main")
}

func foo() int {
	fmt.Println("foo")
	return 3
}

func bar(i int) {
	fmt.Println(i)
}
