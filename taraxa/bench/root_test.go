package bench

import (
	"github.com/Taraxa-project/taraxa-evm/local"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"runtime"
	"runtime/debug"
	"testing"
)

func BenchmarkRoot(b *testing.B) {
	debug.SetGCPercent(-1)
	test_task := func() {
		result := 1
		for i := 2; i < 10000; i++ {
			result *= i
		}
		local.NOOP(result)
	}
	worker_cnt := 24
	task_cnt := worker_cnt * 5000
	b.Run("goroutine_per_task", func(b *testing.B) {
		b.StopTimer()
		for i := 0; i < b.N; i++ {
			tasks_done := concurrent.NewRendezvous(task_cnt)
			b.StartTimer()
			for j := 0; j < task_cnt; j++ {
				go func() {
					test_task()
					tasks_done.CheckIn()
				}()
			}
			tasks_done.Await()
			b.StopTimer()
		}
		runtime.GC()
	})
	b.Run("goroutine_group", func(b *testing.B) {
		b.StopTimer()
		tasks_per_worker := task_cnt / worker_cnt
		for i := 0; i < b.N; i++ {
			//fmt.Println(i)
			tasks_done := concurrent.NewRendezvous(task_cnt)
			b.StartTimer()
			for i := 0; i < worker_cnt; i++ {
				go func() {
					for i := 0; i < tasks_per_worker; i++ {
						test_task()
						tasks_done.CheckIn()
					}
				}()
			}
			tasks_done.Await()
			b.StopTimer()
		}
		runtime.GC()
	})
	b.Run("goroutine_pool", func(b *testing.B) {
		b.StopTimer()
		tasks := make(chan func(), task_cnt)
		all_exited := concurrent.NewRendezvous(worker_cnt)
		for i := 0; i < worker_cnt; i++ {
			go func() {
				defer all_exited.CheckIn()
				for {
					if t, ok := <-tasks; ok {
						t()
						continue
					}
					return
				}
			}()
		}
		for i := 0; i < b.N; i++ {
			//fmt.Println(i)
			tasks_done := concurrent.NewRendezvous(task_cnt)
			b.StartTimer()
			for j := 0; j < task_cnt; j++ {
				tasks <- func() {
					test_task()
					tasks_done.CheckIn()
				}
			}
			tasks_done.Await()
			b.StopTimer()
		}
		close(tasks)
		all_exited.Await()
		runtime.GC()
	})
}
