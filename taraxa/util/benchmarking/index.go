package benchmarking

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"testing"
)

type Benchmark = func(int, *testing.B)

func AddBenchmark(b *testing.B, name string, benchmarkFactory func(string) Benchmark) {
	b.Run(name, func(b *testing.B) {
		b.StopTimer()
		benchmark := benchmarkFactory(name)
		b.StartTimer()
		util.Times(b.N, func(i int) {
			benchmark(i, b)
		})
	})
}
