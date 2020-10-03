package trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type Reader struct{ Schema }

type Proof struct {
	Value []byte
	Nodes [][]byte
}

func (self Reader) Prove(db_tx ReadTxn, root_hash *common.Hash, key *common.Hash) (ret Proof) {
	return
}

func (self Reader) VerifyProof(db_tx ReadTxn, root_hash *common.Hash, key *common.Hash, proof *Proof) bool {
	return true
}

func (self Reader) HashFully(db_tx ReadTxn, root_hash *common.Hash) *common.Hash {
	var kbuf hex_key
	return self.hash_fully(db_tx, (*node_hash)(root_hash), &hash_encoder{}, kbuf[:0]).common_hash()
}

func (self Reader) hash_fully(db_tx ReadTxn, n node, enc *hash_encoder, prefix []byte) (ret *node_hash) {
	is_root := len(prefix) == 0
	switch n := n.(type) {
	case *node_hash:
		return self.hash_fully(db_tx, self.resolve(db_tx, n, prefix), enc, prefix)
	case *short_node:
		hash_list := enc.ListStart()
		enc.AppendString(hex_to_compact(n.key_part, &hex_key_compact{}))
		if val_n, has_val := n.val.(value_node); has_val {
			if val_n == nil_val_node {
				val_n = self.resolve_val_n_by_hex_k(db_tx, append(prefix, n.key_part...))
			}
			_, enc_hash := val_n.val.EncodeForTrie()
			enc.AppendString(enc_hash)
		} else {
			self.hash_fully(db_tx, n.val, enc, append(prefix, n.key_part...))
		}
		enc.ListEnd(hash_list, is_root, &ret)
	case *full_node:
		hash_list := enc.ListStart()
		for i := 0; i < full_node_child_cnt; i++ {
			if c := n.children[i]; c != nil {
				self.hash_fully(db_tx, c, enc, append(prefix, byte(i)))
			} else {
				enc.AppendString(nil)
			}
		}
		enc.AppendString(nil)
		enc.ListEnd(hash_list, is_root, &ret)
	default:
		panic("impossible")
	}
	return
}

type KVCallback = func(*common.Hash, Value)

func (self Reader) ForEach(db_tx ReadTxn, root_hash *common.Hash, with_values bool, cb KVCallback) {
	var kbuf hex_key
	self.for_each(db_tx, (*node_hash)(root_hash), with_values, cb, kbuf[:0])
}

func (self Reader) for_each(db_tx ReadTxn, n node, with_values bool, cb KVCallback, prefix []byte) {
	switch n := n.(type) {
	case *node_hash:
		self.for_each(db_tx, self.resolve(db_tx, n, prefix), with_values, cb, prefix)
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

func (self Reader) resolve(db_tx ReadTxn, hash *node_hash, key_prefix []byte) (ret node) {
	db_tx.GetNode(hash.common_hash(), func(bytes []byte) {
		ret, _ = self.dec_node(db_tx, key_prefix, hash, bytes)
	})
	assert.Holds(ret != nil)
	return
}

func (self Reader) dec_node(db_tx ReadTxn, key_prefix []byte, db_hash *node_hash, buf []byte) (node, []byte) {
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
			panic("impossible")
		}
	case rlp.String:
		switch len(payload) {
		case 0:
			return nil, rest
		case common.HashLength:
			return (*node_hash)(new(common.Hash).SetBytes(payload)), rest
		default:
			panic("impossible")
		}
	default:
		panic("impossible")
	}
}

func (self Reader) dec_short(db_tx ReadTxn, key_prefix []byte, db_hash *node_hash, buf []byte, payload_start byte) *short_node {
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
			assert.Holds(0 < l && l <= self.MaxValueSizeToStoreInTrie())
			ret.val = value_node{internal_value{content, self.ValueStorageToHashEncoding(content)}}
		}
		return ret
	}
	ret.val, _ = self.dec_node(db_tx, append(key_prefix, key_ext...), nil, content)
	if _, child_is_hash := ret.val.(*node_hash); child_is_hash && ret.hash == nil {
		// TODO WTF is this
		ret.hash = (*node_hash)(keccak256.Hash(buf))
	}
	return ret
}

func (self Reader) dec_full(db_tx ReadTxn, key_prefix []byte, db_hash *node_hash, enc []byte) *full_node {
	ret := &full_node{hash: db_hash}
	for i := byte(0); i < full_node_child_cnt; i++ {
		ret.children[i], enc = self.dec_node(db_tx, append(key_prefix, i), nil, enc)
	}
	return ret
}

// TODO lazy load? make sure values are not loaded twice
func (self Reader) resolve_val_n_by_hex_k(db_tx ReadTxn, hex_key []byte) (ret value_node) {
	var key common.Hash
	hex_to_keybytes(hex_key, key[:])
	return self.resolve_val_n(db_tx, &key)
}

func (self Reader) resolve_val_n(db_tx ReadTxn, key *common.Hash) (ret value_node) {
	db_tx.GetValue(key, func(enc_storage []byte) {
		enc_storage = common.CopyBytes(enc_storage)
		ret.val = internal_value{enc_storage, self.ValueStorageToHashEncoding(enc_storage)}
	})
	assert.Holds(ret.val != nil)
	return
}
