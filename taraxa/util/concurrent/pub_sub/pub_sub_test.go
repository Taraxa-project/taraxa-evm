package pub_sub

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/benchmarking"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"sync/atomic"
	"testing"
)

func BenchmarkSumN(b *testing.B) {
	N := 100000
	expectedSum := (N * (N + 1)) / 2
	benchmarking.AddBenchmark(b, "pub_sub", func(b *testing.B, i int) {
		var pubSub PubSub
		reader := pubSub.NewReader()
		quitChan := make(chan interface{})
		go func() {
			total := 0
			for {
				if val := reader.Read(false); val != nil {
					total += val.(int)
				} else {
					break
				}
			}
			util.Assert(total == expectedSum)
			close(quitChan)
		}()
		concurrent.SimpleParallelize(concurrent.CPU_COUNT, N, func(i int) {
			pubSub.Write(i + 1)
		})
		pubSub.RequestTermination()
		<-quitChan
	})
	benchmarking.AddBenchmark(b, "chan", func(b *testing.B, i int) {
		ch := make(chan interface{}, N)
		quitChan := make(chan interface{})
		go func() {
			total := 0
			for {
				if val := <-ch; val != nil {
					total += val.(int)
				} else {
					break
				}
			}
			util.Assert(total == expectedSum)
			close(quitChan)
		}()
		concurrent.SimpleParallelize(concurrent.CPU_COUNT, N, func(i int) {
			ch <- i + 1
		})
		ch <- nil
		<-quitChan
	})
}

func BenchmarkSumNConcurrentReads(b *testing.B) {
	N := 10000000
	expectedSum := uint32((N * (N + 1)) / 2)
	parallelism := concurrent.CPU_COUNT
	benchmarking.AddBenchmark(b, "pub_sub", func(b *testing.B, i int) {
		var pubSub PubSub
		var total uint32
		rendezvous := concurrent.NewRendezvous(parallelism)
		for i := 0; i < parallelism; i++ {
			go func() {
				defer rendezvous.CheckIn()
				reader := pubSub.NewReader()
				for {
					if val := reader.Read(true); val != nil {
						atomic.AddUint32(&total, val.(uint32))
					} else {
						break
					}
				}
			}()
		}
		concurrent.SimpleParallelize(parallelism, N, func(i int) {
			pubSub.Write(uint32(i + 1))
		})
		pubSub.RequestTermination()
		rendezvous.Await()
		total = atomic.LoadUint32(&total)
		util.Assert(total == expectedSum)
	})
	benchmarking.AddBenchmark(b, "chan", func(b *testing.B, i int) {
		ch := make(chan interface{}, N)
		var total uint32
		rendezvous := concurrent.NewRendezvous(parallelism)
		for i := 0; i < parallelism; i++ {
			go func() {
				defer rendezvous.CheckIn()
				for {
					if val, ok := <-ch; ok && val != nil {
						atomic.AddUint32(&total, val.(uint32))
					} else {
						concurrent.TryClose(ch)
						break
					}
				}
			}()
		}
		concurrent.SimpleParallelize(parallelism, N, func(i int) {
			ch <- uint32(i + 1)
		})
		ch <- nil
		rendezvous.Await()
		total = atomic.LoadUint32(&total)
		util.Assert(total == expectedSum)
	})
}
