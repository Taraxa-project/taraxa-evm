package trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
)

func (self *TrieWriter) HashFully() *common.Hash {
	if self.root == nil {
		return nil
	}
	var kbuf hex_key
	return self.hash_fully(self.root.get_hash(), &hash_encoder{}, kbuf[:0]).common_hash()
}

func (self *TrieWriter) hash_fully(n node, enc *hash_encoder, prefix []byte) (ret *node_hash) {
	is_root := len(prefix) == 0
	switch n := n.(type) {
	case *node_hash:
		return self.hash_fully(self.resolve(n, prefix), enc, prefix)
	case *short_node:
		hash_list := enc.ListStart()
		enc.AppendString(hex_to_compact(n.key_part, &hex_key_compact{}))
		if val_n, has_val := n.val.(value_node); has_val {
			if val_n == nil_val_node {
				val_n = self.get_val_node_by_hex_k(append(prefix, n.key_part...))
			}
			_, enc_hash := val_n.val.EncodeForTrie()
			enc.AppendString(enc_hash)
		} else {
			self.hash_fully(n.val, enc, append(prefix, n.key_part...))
		}
		enc.ListEnd(hash_list, is_root, &ret)
		return
	case *full_node:
		hash_list := enc.ListStart()
		for i := 0; i < full_node_child_cnt; i++ {
			if c := n.children[i]; c != nil {
				self.hash_fully(c, enc, append(prefix, byte(i)))
			} else {
				enc.AppendString(nil)
			}
		}
		enc.AppendString(nil)
		enc.ListEnd(hash_list, is_root, &ret)
		return
	}
	panic("impossible")
}
