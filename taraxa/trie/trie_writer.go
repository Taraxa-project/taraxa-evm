package trie

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/emicklei/dot"
	"sync"
)

type TrieWriter struct {
	schema   Schema
	opts     TrieWriterOpts
	root     node
	in       Input
	out      Output
	DotGraph *dot.Graph
	util.InitFlag
}
type TrieWriterOpts struct {
	max_full_nodes_to_keep uint16 // TODO implement
}

const hex_key_len = common.HashLength*2 + 1
const hex_key_compact_len = common.HashLength + 1

type hex_key = [hex_key_len]byte
type hex_key_compact = [hex_key_compact_len]byte

func (self *TrieWriter) I(schema Schema, opts TrieWriterOpts, root_hash *common.Hash) *TrieWriter {
	self.InitOnce()
	self.schema = schema
	self.opts = opts
	if root_hash != nil {
		self.root = (*node_hash)(root_hash)
	}
	return self
}

func (self *TrieWriter) SetIO(in Input, out Output) {
	self.in, self.out = in, out
}

type Value interface {
	EncodeForTrie() (enc_storage, enc_hash []byte)
}

func (self *TrieWriter) Put(k *common.Hash, v Value) {
	assert.Holds(v != nil)
	self.write(k, value_node{v})
}

func (self *TrieWriter) Delete(k *common.Hash) {
	self.write(k, nil_val_node)
}

func (self *TrieWriter) Commit() *common.Hash {
	defer self.SetIO(nil, nil)
	if self.root == nil {
		return nil
	}
	self.root = self.commit(make([]byte, 0, hex_key_len), self.root, nil)
	return self.root.get_hash().common_hash()
}

func (self *TrieWriter) write(k *common.Hash, v value_node) {
	var key_hex hex_key
	keybytes_to_hex(k[:], key_hex[:])
	if v != nil_val_node {
		self.root = self.mpt_insert(self.root, &key_hex, 0, v)
		return
	}
	defer util.Recover(func(issue util.Any) {
		if issue != mpt_del_not_found {
			panic(issue)
		}
	})
	self.root = self.mpt_del(self.root, &key_hex, 0)
	self.out.DeleteValue(k)
}

// TODO maybe dirty checking is worthwhile
func (self *TrieWriter) mpt_insert(n node, key *hex_key, pos int, value value_node) node {
	switch n := n.(type) {
	case *short_node:
		n.hash = nil
		matchlen := prefixLen(key[pos:], n.key_part)
		fork_pos := pos + matchlen
		if fork_pos == len(key) {
			n.val = value
			return n
		}
		if matchlen == len(n.key_part) {
			n.val = self.mpt_insert(n.val, key, fork_pos, value)
			return n
		}
		junction := new(full_node)
		new_pivot := n.key_part[matchlen]
		junction.children[new_pivot] = self.shift(n, key[:fork_pos], new_pivot, n.key_part[matchlen+1:], false)
		junction.children[key[fork_pos]] = &short_node{key_part: key[fork_pos+1:], val: value}
		if matchlen == 0 {
			return junction
		}
		return &short_node{key_part: key[pos:fork_pos], val: junction}
	case *full_node:
		n.hash = nil
		n.children[key[pos]] = self.mpt_insert(n.children[key[pos]], key, pos+1, value)
		return n
	case *node_hash:
		return self.mpt_insert(self.resolve(n, key[:pos]), key, pos, value)
	case nil:
		return &short_node{key_part: key[pos:], val: value}
	}
	panic("impossible")
}

// TODO maybe panic/recover harms performance
var mpt_del_not_found = errors.New("key not found")

func (self *TrieWriter) mpt_del(n node, key *hex_key, pos int) node {
	switch n := n.(type) {
	case *short_node:
		matchlen := prefixLen(key[pos:], n.key_part)
		if matchlen != len(n.key_part) {
			panic(mpt_del_not_found)
		}
		pos += matchlen
		if pos == len(key) {
			return nil
		}
		n.hash = nil
		child := self.mpt_del(n.val, key, pos)
		if short_n, is := child.(*short_node); is {
			n.key_part, n.val = bin.Concat(n.key_part, short_n.key_part...), short_n.val
		} else {
			n.val = child
		}
		return n
	case *full_node:
		deletion_nibble := int8(key[pos])
		deletion_child := self.mpt_del(n.children[deletion_nibble], key, pos+1)
		only_child_nibble := int8(-1)
		if deletion_child == nil {
			for i := int8(0); i < 16; i++ {
				if n.children[i] != nil && i != deletion_nibble {
					if only_child_nibble != -1 {
						only_child_nibble = -1
						break
					}
					only_child_nibble = i
				}
			}
		}
		if only_child_nibble == -1 {
			n.hash = nil
			n.children[deletion_nibble] = deletion_child
			return n
		}
		only_child_nibble_b := byte(only_child_nibble)
		only_child := n.children[only_child_nibble_b]
		if hash_n, is := only_child.(*node_hash); is {
			only_child = self.resolve(hash_n, key[:pos], only_child_nibble_b)
		}
		if short_n, is := only_child.(*short_node); is {
			short_n.hash = nil
			return self.shift(short_n, key[:pos], only_child_nibble_b, short_n.key_part, true)
		}
		return &short_node{key_part: []byte{only_child_nibble_b}, val: only_child}
	case *node_hash:
		return self.mpt_del(self.resolve(n, key[:pos]), key, pos)
	case nil:
		panic(mpt_del_not_found)
	}
	panic("impossible")
}

func (self *TrieWriter) shift(n *short_node, new_prefix []byte, pivot byte, new_suffix []byte, up bool) node {
	// TODO reuse buffers
	if n.val == nil_val_node {
		hex_key := append(make([]byte, 0, len(new_prefix)+1+len(new_suffix)))
		hex_key = append(append(append(hex_key, new_prefix...), pivot), new_suffix...)
		n.val = self.get_val_node_by_hex_k(hex_key)
		if up {
			n.key_part = hex_key[len(new_prefix):]
			return n
		}
	}
	if up {
		n.key_part = append(append(make([]byte, 0, len(new_suffix)+1), pivot), new_suffix...)
		return n
	}
	if len(new_suffix) == 0 {
		return n.val
	}
	n.key_part = new_suffix
	// TODO rehash?
	return n
}

func (self *TrieWriter) resolve(hash *node_hash, key_prefix_base []byte, key_prefix_rest ...byte) (ret node) {
	//cache_miss_cnt.Inc(1) TODO
	enc := self.in.GetNode(hash.common_hash())
	assert.Holds(len(enc) != 0)
	key_prefix := append(append(make([]byte, 0, hex_key_len), key_prefix_base...), key_prefix_rest...)
	ret, _ = self.dec_node(key_prefix, hash, enc)
	return

}

func (self *TrieWriter) dec_node(key_prefix []byte, db_hash *node_hash, buf []byte) (node, []byte) {
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
		case 16:
			return self.dec_full(key_prefix, db_hash, payload), rest
		default:
			panic("impossible")
		}
	case rlp.String:
		switch len(payload) {
		case 0:
			return nil, rest
		case common.HashLength:
			return (*node_hash)(bin.HashView(payload)), rest
		default:
			panic("impossible")
		}
	default:
		panic("impossible")
	}
}

func (self *TrieWriter) dec_short(key_prefix []byte, db_hash *node_hash, enc []byte, payload_start byte) *short_node {
	key_ext, content, err := rlp.SplitString(enc[payload_start:])
	util.PanicIfNotNil(err)
	key_ext = compact_to_hex(key_ext)
	ret := &short_node{key_part: key_ext, hash: db_hash}
	if hasTerm(key_ext) {
		if len(content) == 0 {
			ret.val = self.get_val_node_by_hex_k(append(key_prefix, key_ext...))
			return ret
		}
		content, _, err = rlp.SplitString(content)
		util.PanicIfNotNil(err)
		if l := len(content); l == common.HashLength {
			ret.hash = (*node_hash)(bin.HashView(content))
			ret.val = nil_val_node
		} else {
			assert.Holds(0 < l && l <= self.schema.MaxValueSizeToStoreInTrie())
			ret.val = value_node{RawValue{content, self.schema.ValueStorageToHashEncoding(content)}}
		}
		return ret
	}
	ret.val, _ = self.dec_node(append(key_prefix, key_ext...), nil, content)
	if _, child_is_hash := ret.val.(*node_hash); child_is_hash && ret.hash == nil {
		ret.hash = (*node_hash)(util.Hash(enc))
	}
	return ret
}

func (self *TrieWriter) dec_full(key_prefix []byte, db_hash *node_hash, enc []byte) *full_node {
	ret := &full_node{hash: db_hash}
	for i := byte(0); i < 16; i++ {
		ret.children[i], enc = self.dec_node(append(key_prefix, i), nil, enc)
	}
	return ret
}

type hashing_ctx struct {
	enc_hash            *rlp.Encoder
	enc_storage         *rlp.Encoder
	hasher              *util.Hasher
	hex_key_compact_buf hex_key_compact
	disable_hashing     bool
}

func (self *hashing_ctx) reset() {
	self.enc_hash.Reset()
	self.enc_storage.Reset()
	self.disable_hashing = false
}

func (self *hashing_ctx) toggle_hashing() {
	self.disable_hashing = !self.disable_hashing
}

func (self *hashing_ctx) hash_list_start() *rlp.ListHead {
	if !self.disable_hashing {
		return self.enc_hash.ListStart()
	}
	return nil
}

func (self *hashing_ctx) hash_append_string(buf []byte) {
	if !self.disable_hashing {
		self.enc_hash.AppendString(buf)
	}
}

func (self *hashing_ctx) hash_list_end(out **node_hash, list *rlp.ListHead, is_root bool) {
	if self.disable_hashing {
		return
	}
	self.enc_hash.ListEnd(list)
	if list.Size() < common.HashLength && !is_root {
		return
	}
	self.enc_hash.Flush(list, self.hasher.Write)
	h := self.hasher.Hash()
	util.ReturnHasherToPool(self.hasher)
	self.hasher = util.GetHasherFromPool()
	*out = (*node_hash)(h)
	if !is_root {
		self.enc_hash.EraseSince(list)
		self.enc_hash.AppendString(h[:])
	}
}

var ctx_pool = sync.Pool{New: func() interface{} {
	new_encoder := func() *rlp.Encoder {
		return rlp.NewEncoder(rlp.EncoderConfig{1 << 19, 1 << 8})
	}
	return &hashing_ctx{
		enc_hash:    new_encoder(),
		enc_storage: new_encoder(),
		hasher:      util.GetHasherFromPool(),
	}
}}

// TODO parallel
// TODO cache full nodes
// TODO prettify
func (self *TrieWriter) commit(key_prefix []byte, n node, ctx *hashing_ctx) node {
	dot_draw_level(self.DotGraph, n)
	is_root := ctx == nil
	if is_root {
		if h := n.get_hash(); h != nil {
			return h
		}
		ctx = ctx_pool.Get().(*hashing_ctx)
		defer ctx_pool.Put(ctx)
		defer ctx.reset()
	}
	switch n := n.(type) {
	case *node_hash:
		ctx.hash_append_string(n[:])
		ctx.enc_storage.AppendString(n[:])
		return n
	case *short_node:
		if n.hash != nil {
			ctx.hash_append_string(n.hash[:])
			ctx.toggle_hashing()
			defer ctx.toggle_hashing()
		}
		storage_list := ctx.enc_storage.ListStart()
		hash_list := ctx.hash_list_start()
		hashed_key_ext := hex_to_compact(n.key_part, &ctx.hex_key_compact_buf)
		ctx.enc_storage.AppendString(hashed_key_ext)
		ctx.hash_append_string(hashed_key_ext)
		key_extended := append(key_prefix, n.key_part...)
		val_n, has_val := n.val.(value_node)
		var val RawValue
		if has_val {
			if val_n != nil_val_node {
				val.ENC_storage, val.ENC_hash = val_n.EncodeForTrie()
				val_n.Value = val
				var key common.Hash
				hex_to_keybytes(key_extended, key[:])
				self.out.PutValue(&key, val.ENC_storage)
			} else {
				assert.Holds(n.hash != nil)
			}
			if !ctx.disable_hashing {
				ctx.hash_append_string(val.ENC_hash)
			}
		} else {
			n.val = self.commit(key_extended, n.val, ctx)
		}
		ctx.hash_list_end(&n.hash, hash_list, is_root)
		if has_val {
			if n.hash != nil {
				ctx.enc_storage.AppendString(n.hash[:])
			} else if len(val.ENC_storage) <= self.schema.MaxValueSizeToStoreInTrie() {
				ctx.enc_storage.AppendString(val.ENC_storage)
			}
		}
		ctx.enc_storage.ListEnd(storage_list)
		if is_root {
			self.out.PutNode(n.hash.common_hash(), ctx.enc_storage.ToBytes(storage_list))
			return n.hash
		}
		return n
	case *full_node:
		if n.hash != nil {
			ctx.hash_append_string(n.hash[:])
			ctx.enc_storage.AppendString(n.hash[:])
			return n.hash
		}
		hash_list := ctx.hash_list_start()
		storage_list := ctx.enc_storage.ListStart()
		for i := byte(0); i < 16; i++ {
			if c := n.children[i]; c != nil {
				n.children[i] = self.commit(append(key_prefix, i), c, ctx)
			} else {
				ctx.hash_append_string(nil)
				ctx.enc_storage.AppendEmptyString()
			}
		}
		ctx.enc_storage.ListEnd(storage_list)
		ctx.hash_append_string(nil)
		ctx.hash_list_end(&n.hash, hash_list, is_root)
		if n.hash != nil {
			self.out.PutNode(n.hash.common_hash(), ctx.enc_storage.ToBytes(storage_list))
			if !is_root {
				ctx.enc_storage.EraseSince(storage_list)
				ctx.enc_storage.AppendString(n.hash[:])
			}
			return n.hash
		}
		return n
	}
	panic("impossible")
}

func (self *TrieWriter) get_val_node_by_hex_k(hex_key []byte) value_node {
	var key common.Hash
	hex_to_keybytes(hex_key, key[:])
	enc_storage := self.in.GetValue(&key)
	assert.Holds(len(enc_storage) != 0)
	return value_node{RawValue{enc_storage, self.schema.ValueStorageToHashEncoding(enc_storage)}}
}

var nil_val_node = value_node{nil}
