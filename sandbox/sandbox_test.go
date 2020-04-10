package main

import (
	"testing"
)

func BenchmarkRoot(b *testing.B) {
	b.Run("1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f1()
		}
	})
	b.Run("2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			f2()
		}
	})

}
