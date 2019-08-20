package main

import "sync/atomic"

type F = func(string) int

func main() {
	var f atomic.Value
	f.Store(func(s string) int {
		return 1
	})
	f.Load().(F)("foo")
}
