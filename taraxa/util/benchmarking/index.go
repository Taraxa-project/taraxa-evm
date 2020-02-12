package benchmarking

import (
	"runtime"
	"runtime/debug"
	"testing"
)

type Benchmark = func(b *testing.B, i int)

func AddBenchmark(b *testing.B, name string, benchmark Benchmark) {
	b.Run(name, func(b *testing.B) {
		b.StopTimer()
		prev_gc_pct := debug.SetGCPercent(-1)
		defer debug.SetGCPercent(prev_gc_pct)
		var max_heap_size uint64
		for i := 0; i < b.N; i++ {
			if prev_gc_pct > 0 {
				var mem_stats runtime.MemStats
				if runtime.ReadMemStats(&mem_stats); max_heap_size == 0 || mem_stats.HeapAlloc > max_heap_size {
					if max_heap_size != 0 {
						runtime.GC()
						runtime.ReadMemStats(&mem_stats)
					}
					max_heap_size = (mem_stats.HeapAlloc * uint64(prev_gc_pct)) / 100
				}
			}
			b.StartTimer()
			benchmark(b, i)
			b.StopTimer()
		}
	})
}
