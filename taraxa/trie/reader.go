package trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type Reader struct{ ReadOnlyDB }

type Proof struct {
	Value []byte
	Nodes [][]byte
}

func (self Reader) Prove(root_hash *common.Hash, key *common.Hash) (ret Proof) {
	return
}

func (self Reader) VerifyProof(root_hash *common.Hash, key *common.Hash, proof *Proof) bool {
	return true
}

func (self Reader) HashFully(root_hash *common.Hash) *common.Hash {
	var kbuf hex_key
	return self.hash_fully((*node_hash)(root_hash), &hash_encoder{}, kbuf[:0]).common_hash()
}

func (self Reader) hash_fully(n node, enc *hash_encoder, prefix []byte) (ret *node_hash) {
	is_root := len(prefix) == 0
	switch n := n.(type) {
	case *node_hash:
		return self.hash_fully(self.resolve(n, prefix), enc, prefix)
	case *short_node:
		hash_list := enc.ListStart()
		enc.AppendString(hex_to_compact(n.key_part, &hex_key_compact{}))
		if val_n, has_val := n.val.(value_node); has_val {
			if val_n == nil_val_node {
				val_n = self.resolve_val_n_by_hex_k(append(prefix, n.key_part...))
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

func (self Reader) resolve(hash *node_hash, key_prefix []byte) (ret node) {
	enc := self.GetNode(hash.common_hash())
	assert.Holds(len(enc) != 0)
	ret, _ = self.dec_node(key_prefix, hash, enc)
	return
}

func (self Reader) dec_node(key_prefix []byte, db_hash *node_hash, buf []byte) (node, []byte) {
	kind, tagsize, total_size, err := rlp.ReadKind(buf)
	util.PanicIfNotNil(err)
	payload, rest := buf[tagsize:total_size], buf[total_size:]
	switch kind {
	case rlp.List:
		size, err := rlp.CountValues(payload) // TODO optimize
		util.PanicIfNotNil(err)
		switch size {
		case 1, 2:
			return self.dec_short(key_prefix, db_hash, buf[:total_size], tagsize), rest
		case full_node_child_cnt:
			return self.dec_full(key_prefix, db_hash, payload), rest
		default:
			panic("impossible")
		}
	case rlp.String:
		switch len(payload) {
		case 0:
			return nil, rest
		case common.HashLength:
			return (*node_hash)(keccak256.HashView(payload)), rest
		default:
			panic("impossible")
		}
	default:
		panic("impossible")
	}
}

func (self Reader) dec_short(key_prefix []byte, db_hash *node_hash, enc []byte, payload_start byte) *short_node {
	key_ext, content, err := rlp.SplitString(enc[payload_start:])
	util.PanicIfNotNil(err)
	key_ext = compact_to_hex(key_ext)
	ret := &short_node{key_part: key_ext, hash: db_hash}
	if hasTerm(key_ext) {
		if len(content) == 0 {
			ret.val = self.resolve_val_n_by_hex_k(append(key_prefix, key_ext...))
			return ret
		}
		content, _, err = rlp.SplitString(content)
		util.PanicIfNotNil(err)
		if l := len(content); l == common.HashLength {
			ret.hash = (*node_hash)(keccak256.HashView(content))
			ret.val = nil_val_node
		} else {
			assert.Holds(0 < l && l <= self.MaxValueSizeToStoreInTrie())
			ret.val = value_node{internal_value{content, self.ValueStorageToHashEncoding(content)}}
		}
		return ret
	}
	ret.val, _ = self.dec_node(append(key_prefix, key_ext...), nil, content)
	if _, child_is_hash := ret.val.(*node_hash); child_is_hash && ret.hash == nil {
		ret.hash = (*node_hash)(keccak256.Hash(enc))
	}
	return ret
}

func (self Reader) dec_full(key_prefix []byte, db_hash *node_hash, enc []byte) *full_node {
	ret := &full_node{hash: db_hash}
	for i := byte(0); i < full_node_child_cnt; i++ {
		ret.children[i], enc = self.dec_node(append(key_prefix, i), nil, enc)
	}
	return ret
}

func (self Reader) resolve_val_n_by_hex_k(hex_key []byte) (ret value_node) {
	var key common.Hash
	hex_to_keybytes(hex_key, key[:])
	enc_storage := self.GetValue(&key)
	assert.Holds(len(enc_storage) != 0)
	return value_node{internal_value{enc_storage, self.ValueStorageToHashEncoding(enc_storage)}}
}
