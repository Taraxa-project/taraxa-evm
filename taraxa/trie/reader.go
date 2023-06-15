package trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type Reader struct{ Schema }
type KVCallback = func(*common.Hash, Value)

func (self Reader) ForEach(db_tx Input, root_hash *common.Hash, with_values bool, cb KVCallback) {
	var kbuf hex_key
	self.for_each(db_tx, (*node_hash)(root_hash), with_values, cb, kbuf[:0])
}

func (self Reader) ForEachNodeHash(db_tx Input, root_hash *common.Hash, cb func(*common.Hash, []byte)) {
	var kbuf hex_key
	self.for_each_node_hash(db_tx, (*node_hash)(root_hash), cb, kbuf[:0])
}

func (self Reader) for_each_node_hash(db_tx Input, n node, cb func(*common.Hash, []byte), prefix []byte) {
	switch n := n.(type) {
	case *node_hash:
		ret_node, ret_bytes := self.resolve(db_tx, n, prefix)
		cb(n.common_hash(), ret_bytes)
		self.for_each_node_hash(db_tx, ret_node, cb, prefix)
	case *short_node:
		key_extended := append(prefix, n.key_part...)
		if _, has_val := n.val.(value_node); !has_val {
			self.for_each_node_hash(db_tx, n.val, cb, key_extended)
		}
	case *full_node:
		for i := 0; i < full_node_child_cnt; i++ {
			if c := n.children[i]; c != nil {
				self.for_each_node_hash(db_tx, c, cb, append(prefix, byte(i)))
			}
		}
	default:
		panic("impossible")
	}
}

func (self Reader) for_each(db_tx Input, n node, with_values bool, cb KVCallback, prefix []byte) {
	switch n := n.(type) {
	case *node_hash:
		ret_node, _ := self.resolve(db_tx, n, prefix)
		self.for_each(db_tx, ret_node, with_values, cb, prefix)
	case *short_node:
		key_extended := append(prefix, n.key_part...)
		if val_n, has_val := n.val.(value_node); has_val {
			var key common.Hash
			hex_to_keybytes(key_extended, key[:])
			if val_n == nil_val_node && with_values {
				val_n = self.resolve_val_n(db_tx, &key)
			}
			cb(&key, val_n.val)
		} else {
			self.for_each(db_tx, n.val, with_values, cb, key_extended)
		}
	case *full_node:
		for i := 0; i < full_node_child_cnt; i++ {
			if c := n.children[i]; c != nil {
				self.for_each(db_tx, c, with_values, cb, append(prefix, byte(i)))
			}
		}
	default:
		panic("impossible")
	}
}

func (self Reader) resolve(db_tx Input, hash *node_hash, key_prefix []byte) (ret node, ret_bytes []byte) {
	db_tx.GetNode(hash.common_hash(), func(bytes []byte) {
		ret, _ = self.dec_node(db_tx, key_prefix, hash, bytes)
		ret_bytes = common.CopyBytes(bytes)
	})
	asserts.Holds(ret != nil)
	return
}

func (self Reader) dec_node(db_tx Input, key_prefix []byte, db_hash *node_hash, buf []byte) (node, []byte) {
	kind, tagsize, total_size, err := rlp.ReadKind(buf)
	util.PanicIfNotNil(err)
	payload, rest := buf[tagsize:total_size], buf[total_size:]
	switch kind {
	case rlp.List:
		size, err := rlp.CountValues(payload) // TODO optimize
		util.PanicIfNotNil(err)
		switch size {
		case 1, 2:
			return self.dec_short(db_tx, key_prefix, db_hash, buf[:total_size], tagsize), rest
		case full_node_child_cnt:
			return self.dec_full(db_tx, key_prefix, db_hash, payload), rest
		default:
			panic("impossible " + db_hash.common_hash().Hex())
		}
	case rlp.String:
		switch len(payload) {
		case 0:
			return nil, rest
		case common.HashLength:
			return (*node_hash)(new(common.Hash).SetBytes(payload)), rest
		default:
			panic("impossible " + db_hash.common_hash().Hex())
		}
	default:
		panic("impossible " + db_hash.common_hash().Hex())
	}
}

func (self Reader) dec_short(db_tx Input, key_prefix []byte, db_hash *node_hash, buf []byte, payload_start byte) *short_node {
	key_ext, content, err := rlp.SplitString(buf[payload_start:])
	util.PanicIfNotNil(err)
	key_ext = compact_to_hex(key_ext)
	ret := &short_node{key_part: key_ext, hash: db_hash}
	if hasTerm(key_ext) {
		if len(content) == 0 {
			ret.val = self.resolve_val_n_by_hex_k(db_tx, append(key_prefix, key_ext...))
			return ret
		}
		content, _, err = rlp.SplitString(content)
		util.PanicIfNotNil(err)
		content = common.CopyBytes(content)
		if l := len(content); l == common.HashLength {
			ret.hash = (*node_hash)(keccak256.HashView(content))
			ret.val = nil_val_node
		} else {
			asserts.Holds(0 < l && l <= self.MaxValueSizeToStoreInTrie())
			ret.val = value_node{internal_value{content, self.ValueStorageToHashEncoding(content)}}
		}
		return ret
	}
	ret.val, _ = self.dec_node(db_tx, append(key_prefix, key_ext...), nil, content)
	if _, child_is_hash := ret.val.(*node_hash); child_is_hash && ret.hash == nil {
		ret.hash = (*node_hash)(keccak256.Hash(buf))
	}
	return ret
}

func (self Reader) dec_full(db_tx Input, key_prefix []byte, db_hash *node_hash, enc []byte) *full_node {
	ret := &full_node{hash: db_hash}
	for i := byte(0); i < full_node_child_cnt; i++ {
		ret.children[i], enc = self.dec_node(db_tx, append(key_prefix, i), nil, enc)
	}
	return ret
}

// TODO lazy load? make sure values are not loaded twice
func (self Reader) resolve_val_n_by_hex_k(db_tx Input, hex_key []byte) (ret value_node) {
	var key common.Hash
	hex_to_keybytes(hex_key, key[:])
	return self.resolve_val_n(db_tx, &key)
}

func (self Reader) resolve_val_n(db_tx Input, key *common.Hash) (ret value_node) {
	db_tx.GetValue(key, func(enc_storage []byte) {
		enc_storage = common.CopyBytes(enc_storage)
		ret.val = internal_value{enc_storage, self.ValueStorageToHashEncoding(enc_storage)}
	})
	asserts.Holds(ret.val != nil)
	return
}
