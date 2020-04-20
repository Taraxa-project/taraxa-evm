package trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"runtime"
	"sync"
)

const commit_ctx_min_depth_to_cache = 8

var commit_ctx_pools = func() (ret [MaxDepth - commit_ctx_min_depth_to_cache + 1]*sync.Pool) {
	for i := 0; i < len(ret); i++ {
		depth := commit_ctx_min_depth_to_cache + i
		ret[i] = &sync.Pool{New: func() interface{} {
			ret := new(commit_context)
			hash_str_buf_cap := depth * full_node_child_cnt * common.HashLength / 4
			ret.enc_hash.ResizeReset(hash_str_buf_cap, depth)
			ret.enc_storage.ResizeReset(int(float64(hash_str_buf_cap)*1.2), depth)
			return ret
		}}
	}
	return
}()

func get_commit_ctx(depth byte) *commit_context {
	if depth < commit_ctx_min_depth_to_cache {
		return new(commit_context)
	}
	pool := commit_ctx_pools[depth-commit_ctx_min_depth_to_cache]
	ret := pool.Get().(*commit_context)
	ret.Reset()
	runtime.SetFinalizer(ret, func(ctx *commit_context) {
		if ctx.enc_hash.encoder.ListsCount() == int(depth) {
			pool.Put(ctx)
		}
	})
	return ret
}

type commit_context struct {
	hex_key_compact_tmp hex_key_compact
	enc_hash            hash_encoder
	enc_storage         rlp.Encoder
}

func (self *commit_context) Reset() {
	self.enc_hash.Reset()
	self.enc_storage.Reset()
}
