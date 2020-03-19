package trie

import "github.com/Taraxa-project/taraxa-evm/metrics"

var cache_miss_cnt = metrics.NewRegisteredCounter("trie/cachemiss", nil)
var cache_unload_cnt = metrics.NewRegisteredCounter("trie/cacheunload", nil)

func CacheMisses() int64 {
	return cache_miss_cnt.Count()
}

func CacheUnloads() int64 {
	return cache_unload_cnt.Count()
}
