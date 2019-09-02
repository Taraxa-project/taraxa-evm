package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/rand"
	"runtime/debug"
	"strconv"
	"sync"
	"testing"
)

func init() {
	debug.SetGCPercent(-1)
}

var keys = func() []common.Address {
	ret := make([]common.Address, 100000)
	for i := 0; i < len(ret); i++ {
		b := new(bytes.Buffer)
		for i := 0; i < 5; i++ {
			binary.Write(b, binary.LittleEndian, rand.Uint32())
		}
		ret[i] = common.BytesToAddress(b.Bytes())
	}
	return ret
}()

type shard struct {
	data map[common.Address]interface{}
	mu   sync.RWMutex
}

type ShardedMap struct {
	shardCount uint64
	shards     []*shard
}

func NewShardedMap(shardCount uint64) *ShardedMap {
	this := &ShardedMap{shardCount, make([]*shard, shardCount)}
	for i := uint64(0); i < shardCount; i++ {
		this.shards[i] = &shard{data: make(map[common.Address]interface{})}
	}
	return this
}

func (this *ShardedMap) getShard(k common.Address) *shard {
	return this.shards[util.CRC64(k[:])%this.shardCount]
}

func (this *ShardedMap) Get(k common.Address) (val interface{}, ok bool) {
	shard := this.getShard(k)
	defer concurrent.LockUnlock(shard.mu.RLocker())()
	val, ok = shard.data[k]
	return
}

func (this *ShardedMap) Put(k common.Address, v interface{}) (prev interface{}, prevHasBeen bool) {
	shard := this.getShard(k)
	defer concurrent.LockUnlock(&shard.mu)()
	prev, prevHasBeen = shard.data[k]
	shard.data[k] = v
	return
}

//func BenchmarkCHMPut(b *testing.B) {
//	for i := 0; i < b.N; i++ {
//		b.StopTimer()
//		m := new(taraxa.ConcurrentHashMap)
//		b.StartTimer()
//		concurrent.Parallelize(parallelism, len(keys), func(int) func(int) {
//			return func(i int) {
//				m.Insert(keys[i].String(), 1)
//			}
//		})
//	}
//}
//
//func BenchmarkCHMGet(b *testing.B) {
//	b.StopTimer()
//	m := new(taraxa.ConcurrentHashMap)
//	for _, k := range keys {
//		m.Insert(k.String(), 1)
//	}
//	b.StartTimer()
//	for i := 0; i < b.N; i++ {
//		concurrent.Parallelize(parallelism, len(keys), func(int) func(int) {
//			return func(i int) {
//				m.Get(keys[i].String())
//			}
//		})
//	}
//}

func BenchmarkShardedMapPut(b *testing.B) {
	for _, shardCount := range []int{8, 16, 32, 64, 128, 256} {
		for _, parallelism := range []int{4, 8, 12, 16, 20, 24, 32} {
			b.Run(fmt.Sprintf("sharded_map_put_%s_%s", strconv.Itoa(shardCount), strconv.Itoa(parallelism)),
				func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						m := NewShardedMap(uint64(shardCount))
						b.StartTimer()
						concurrent.Parallelize(parallelism, len(keys), func(int) func(int) {
							return func(i int) {
								m.Put(keys[i], 1)
							}
						})
					}
				})
		}
	}
}

func BenchmarkShardedMapGet(b *testing.B) {
	for _, shardCount := range []int{8, 16, 32, 64, 128, 256} {
		for _, parallelism := range []int{4, 8, 12, 16, 20, 24, 32} {
			b.Run(fmt.Sprintf("sharded_map_get_%s_%s", strconv.Itoa(shardCount), strconv.Itoa(parallelism)),
				func(b *testing.B) {
					b.StopTimer()
					m := NewShardedMap(uint64(shardCount))
					for _, k := range keys {
						m.Put(k, 1)
					}
					b.StartTimer()
					for i := 0; i < b.N; i++ {
						concurrent.Parallelize(parallelism, len(keys), func(int) func(int) {
							return func(i int) {
								m.Get(keys[i])
							}
						})
					}
				})
		}
	}
}
