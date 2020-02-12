package main

import (
	"fmt"
)

type FN = func(int) string

func main() {
	var foo interface{} = func(i int) string {
		return "foo"
	}
	switch f := foo.(type) {
	case FN:
		fmt.Println(f(1))
	}
}
