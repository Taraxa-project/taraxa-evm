package main

import "fmt"

type s struct {
	m map[string]int
}

func main() {
	var m = make(map[string]int, 3)
	m["ff"] = 1
	fmt.Println(m["ff"])
}
