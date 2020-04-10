package trie

import "github.com/Taraxa-project/taraxa-evm/common"

func (self *TrieWriter) HashFully(root_hash *common.Hash) *common.Hash {
	if root_hash == nil {
		return nil
	}
	return self.hash_fully((*node_hash)(root_hash), nil, make([]byte, 0, hex_key_len)).common_hash()
}

func (self *TrieWriter) hash_fully(n node, ctx *hashing_ctx, prefix []byte) *node_hash {
	is_root := ctx == nil
	if is_root {
		ctx = ctx_pool.Get().(*hashing_ctx)
		defer ctx_pool.Put(ctx)
		defer ctx.reset()
	}
	switch n := n.(type) {
	case *node_hash:
		return self.hash_fully(self.resolve(n, prefix), ctx, prefix)
	case *short_node:
		hash_list := ctx.hash_list_start()
		hashed_key_ext := hex_to_compact(n.key_part, &ctx.hex_key_compact_buf)
		ctx.hash_append_string(hashed_key_ext)
		val, has_val := n.val.(value_node)
		if has_val {
			if !ctx.disable_hashing {
				if val.Value == nil {
					val = self.get_val_node_by_hex_k(append(prefix, n.key_part...))
				}
				_, enc_hash := val.EncodeForTrie()
				ctx.hash_append_string(enc_hash)
			}
		} else {
			self.hash_fully(n.val, ctx, append(prefix, n.key_part...))
		}
		var h *node_hash
		ctx.hash_list_end(&h, hash_list, is_root)
		return h
	case *full_node:
		hash_list := ctx.hash_list_start()
		for i := 0; i < 16; i++ {
			if c := n.children[i]; c != nil {
				self.hash_fully(c, ctx, append(prefix, byte(i)))
			} else {
				ctx.hash_append_string(nil)
			}
		}
		ctx.hash_append_string(nil)
		var h *node_hash
		ctx.hash_list_end(&h, hash_list, is_root)
		return h
	default:
		panic("impossible")
	}
}
