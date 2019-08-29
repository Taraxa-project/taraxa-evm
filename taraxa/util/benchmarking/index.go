package benchmarking

import (
	"testing"
)

type Benchmark = func(b *testing.B, i int, name string)

func AddBenchmark(b *testing.B, name string, benchmark Benchmark) {
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchmark(b, i, name)
		}
	})
}
