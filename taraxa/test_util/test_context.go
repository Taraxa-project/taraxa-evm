package test_util

import (
	"reflect"
	"runtime"
	"runtime/debug"
	"testing"
)

type TestContext struct {
	t             *testing.T
	b             *testing.B
	bench_targets []function
	test_targets  []function
	benchmarks    []function
	tests         []function
}

type BenchContext struct {
	*testing.B
	DisableGC bool
}
type function = struct {
	name string
	fn   reflect.Value
}

func NewTestingContext(ctx testing.TB) (self *TestContext) {
	self = new(TestContext)
	self.t, _ = ctx.(*testing.T)
	self.b, _ = ctx.(*testing.B)
	return
}

func RunTests(tb testing.TB, setup func(ctx *TestContext)) {
	ctx := NewTestingContext(tb)
	setup(ctx)
	ctx.Run()
}

func (self *TestContext) TARGET_TEST(name string, fn interface{}) {
	if self.t != nil {
		self.test_targets = append(self.test_targets, function{name, reflect.ValueOf(fn)})
	}
}

func (self *TestContext) TARGET_BENCH(name string, fn interface{}) {
	if self.b != nil {
		self.bench_targets = append(self.bench_targets, function{name, reflect.ValueOf(fn)})
	}
}

func (self *TestContext) TARGET_ALL(name string, fn interface{}) {
	if self.t != nil {
		self.test_targets = append(self.test_targets, function{name, reflect.ValueOf(fn)})
	}
	if self.b != nil {
		self.bench_targets = append(self.bench_targets, function{name, reflect.ValueOf(fn)})
	}
}

func (self *TestContext) BENCH(name string, fn interface{}) {
	if self.b != nil {
		self.benchmarks = append(self.benchmarks, function{name, reflect.ValueOf(fn)})
	}

}

func (self *TestContext) TEST(name string, fn interface{}) {
	if self.t != nil {
		self.tests = append(self.tests, function{name, reflect.ValueOf(fn)})
	}
}

func (self *TestContext) Run() {
	if self.t != nil {
		for _, test := range self.tests {
			if test.fn.Type().NumIn() == 1 {
				self.t.Run(test.name, func(t *testing.T) {
					test.fn.Call([]reflect.Value{
						reflect.ValueOf(t),
					})
				})
				continue
			}
			for _, target := range self.test_targets {
				self.t.Run(test.name+"/"+target.name, func(t *testing.T) {
					reflect_ctx := reflect.ValueOf(t)
					reflect_sut := target.fn.Call([]reflect.Value{
						reflect_ctx,
					})[0]
					test.fn.Call([]reflect.Value{
						reflect_ctx,
						reflect_sut,
					})
				})
			}
		}
	}
	if self.b == nil {
		return
	}
	for _, bench := range self.benchmarks {
		if bench.fn.Type().NumIn() == 1 {
			self.b.Run(bench.name, func(b *testing.B) {
				bench.fn.Call([]reflect.Value{
					reflect.ValueOf(b),
				})
			})
			continue
		}
		for _, target := range self.bench_targets {
			self.b.Run(bench.name+"/"+target.name, func(b *testing.B) {
				b.StopTimer()
				ctx := &BenchContext{B: b}
				reflect_ctx := reflect.ValueOf(ctx)
				reflect_sut := target.fn.Call([]reflect.Value{
					reflect_ctx,
				})[0]
				var prev_gc_pct int
				var max_heap_size uint64
				if ctx.DisableGC {
					prev_gc_pct = debug.SetGCPercent(-1)
					defer debug.SetGCPercent(prev_gc_pct)
				}
				for i := 0; i < b.N; i++ {
					if ctx.DisableGC {
						// TODO enable/disable during benchmark
						if prev_gc_pct > 0 {
							var mem_stats runtime.MemStats
							runtime.ReadMemStats(&mem_stats)
							if max_heap_size == 0 || mem_stats.HeapAlloc > max_heap_size {
								if max_heap_size != 0 {
									runtime.GC()
									runtime.ReadMemStats(&mem_stats)
								}
								max_heap_size = (mem_stats.HeapAlloc * uint64(prev_gc_pct)) / 100
							}
						}
					}
					b.StartTimer()
					bench.fn.Call([]reflect.Value{
						reflect_ctx,
						reflect_sut,
						reflect.ValueOf(i),
					})
					b.StopTimer()
				}
			})
		}
	}
}
