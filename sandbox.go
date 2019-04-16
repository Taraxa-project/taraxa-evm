package main

import "fmt"

type OperationType int

const (
	GET OperationType = iota
	SET
	ADD
	OperationType_count uint = iota
)
func main() {
	a := make([]int, OperationType_count)
	fmt.Println(a[3])
}
