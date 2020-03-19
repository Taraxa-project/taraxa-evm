package trie

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/emicklei/dot"
	"sync"
)

type Trie struct {
	root                   node
	schema                 Schema
	storage                Storage
	max_full_nodes_to_keep uint16
	DotGraph               *dot.Graph
}
type Schema = interface {
	FlatKey(hashed_key []byte) (flat_key []byte)
	StorageToHashEncoding(enc_storage []byte) (enc_hash []byte)
	MaxStorageEncSizeToStoreInTrie() int
}
type Storage interface {
	PutAsync(col StorageColumn, key, value []byte)
	DeleteAsync(col StorageColumn, key []byte)
	GetCommitted(col StorageColumn, key []byte) []byte
}
type StorageColumn = byte

const (
	COL_flat_key_to_value StorageColumn = iota
	COL_hash_to_node
)

func NewTrie(root_hash []byte, schema Schema, storage Storage, max_full_nodes_to_keep uint16) *Trie {
	ret := &Trie{
		schema:                 schema,
		storage:                storage,
		max_full_nodes_to_keep: max_full_nodes_to_keep,
	}
	if len(root_hash) != 0 {
		ret.root = node_hash(root_hash)
	}
	return ret
}

func (self *Trie) Get(key []byte) (enc_storage []byte) {
	return self.get_val_enc_stoage_by_hashed_k(util.Keccak256Pooled(key))
}

func (self *Trie) Put(k, enc_storage, enc_hash []byte) {
	self.update(k, &value{enc_storage, enc_hash})
}

func (self *Trie) Delete(k []byte) {
	self.update(k, nil)
}

func (self *Trie) CommitNodes() []byte {
	if self.root == nil {
		return nil
	}
	self.root = self.commit(self.root, nil)
	return self.root.get_hash()
}

//func (self *Trie) mpt_get(origNode node, key_hex []byte, pos int) (value []byte, newnode node, didResolve bool) {
//	switch n := (origNode).(type) {
//	case nil:
//		return nil, nil, false
//	case value:
//		return n, n, false
//	case *short_node:
//		if len(key_hex)-pos < len(n.key_part) || !bytes.Equal(n.key_part, key_hex[pos:pos+len(n.key_part)]) {
//			// key not found in trie
//			return nil, n, false
//		}
//		value, newnode, didResolve = self.mpt_get(n.val, key_hex, pos+len(n.key_part))
//		if didResolve {
//			n = n.copy()
//			n.val = newnode
//			n.node_status.gen = self.cachegen
//		}
//		return value, n, didResolve
//	case *full_node:
//		value, newnode, didResolve = self.mpt_get(n.children[key_hex[pos]], key_hex, pos+1)
//		if didResolve {
//			n = n.copy()
//			n.node_status.gen = self.cachegen
//			n.children[key_hex[pos]] = newnode
//		}
//		return value, n, didResolve
//	case node_hash:
//		val := self.resolve(n, key_hex[:pos])
//		value, newnode, _ := self.mpt_get(val, key_hex, pos)
//		return value, newnode, true
//	default:
//		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
//	}
//}

func (self *Trie) update(key []byte, val *value) {
	mpt_key := util.Keccak256Pooled(key)
	flat_key := self.schema.FlatKey(mpt_key)
	mpt_key_hex := keybytesToHex(mpt_key)
	if val != nil {
		self.root = self.mpt_insert(self.root, mpt_key_hex, 0, val)
		self.storage.PutAsync(COL_flat_key_to_value, flat_key, val.enc_storage)
		return
	}
	defer func() {
		if rec := recover(); rec != nil && rec != mpt_del_not_found {
			panic(rec)
		}
	}()
	self.root = self.mpt_del(self.root, mpt_key_hex, 0)
	self.storage.DeleteAsync(COL_flat_key_to_value, flat_key)
}

// TODO maybe dirty checking is worthwhile
func (self *Trie) mpt_insert(n node, key []byte, pos int, value *value) node {
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
	case node_hash:
		return self.mpt_insert(self.resolve(n, key[:pos]), key, pos, value)
	case nil:
		return &short_node{key_part: key[pos:], val: value}
	}
	panic("impossible")
}

// TODO maybe panic/recover harms performance
var mpt_del_not_found = errors.New("key not found")

func (self *Trie) mpt_del(n node, key []byte, pos int) node {
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
			n.key_part, n.val = binary.Concat(n.key_part, short_n.key_part...), short_n.val
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
		if hash_n, is := only_child.(node_hash); is {
			only_child = self.resolve(hash_n, key[:pos], only_child_nibble_b)
		}
		if short_n, is := only_child.(*short_node); is {
			short_n.hash = nil
			return self.shift(short_n, key[:pos], only_child_nibble_b, short_n.key_part, true)
		}
		return &short_node{key_part: []byte{only_child_nibble_b}, val: only_child}
	case node_hash:
		return self.mpt_del(self.resolve(n, key[:pos]), key, pos)
	case nil:
		panic(mpt_del_not_found)
	}
	panic("impossible")
}

func (self *Trie) shift(n *short_node, new_prefix []byte, pivot byte, new_suffix []byte, up bool) node {
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

func (self *Trie) resolve(hash node_hash, key_prefix []byte, key_prefix_rest ...byte) node {
	//cache_miss_cnt.Inc(1) TODO
	enc := self.storage.GetCommitted(COL_hash_to_node, hash)
	// TODO reuse
	key_prefix_buf := append(append(make([]byte, 0, common.HashLength*2+1), key_prefix...), key_prefix_rest...)
	ret, _ := self.dec_node(key_prefix_buf, hash, enc)
	return ret

}

func (self *Trie) dec_node(key_prefix, db_hash, buf []byte) (node, []byte) {
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
			return node_hash(payload), rest
		default:
			panic("impossible")
		}
	default:
		panic("impossible")
	}
}

func (self *Trie) dec_short(key_prefix, db_hash, enc []byte, payload_start byte) *short_node {
	key_ext, content, err := rlp.SplitString(enc[payload_start:])
	util.PanicIfNotNil(err)
	key_ext = compactToHex(key_ext)
	ret := &short_node{key_part: key_ext, hash: db_hash}
	if hasTerm(key_ext) {
		if len(content) == 0 {
			ret.val = self.get_val_node_by_hex_k(append(key_prefix, key_ext...))
			return ret
		}
		content, _, err = rlp.SplitString(content)
		util.PanicIfNotNil(err)
		if l := len(content); l == common.HashLength {
			ret.hash = content
			ret.val = nil_val_node
		} else {
			util.Assert(0 < l && l <= self.schema.MaxStorageEncSizeToStoreInTrie())
			ret.val = &value{content, self.schema.StorageToHashEncoding(content)}
		}
		return ret
	}
	ret.val, _ = self.dec_node(append(key_prefix, key_ext...), nil, content)
	if _, child_is_hash := ret.val.(node_hash); child_is_hash && len(ret.hash) == 0 {
		ret.hash = util.Keccak256Pooled(enc)
	}
	return ret
}

func (self *Trie) dec_full(key_prefix, db_hash, enc []byte) *full_node {
	ret := &full_node{hash: db_hash}
	for i := byte(0); i < 16; i++ {
		ret.children[i], enc = self.dec_node(append(key_prefix, i), nil, enc)
	}
	return ret
}

type hashing_ctx struct {
	enc_hash           *rlp.Encoder
	enc_storage        *rlp.Encoder
	hasher             *util.Hasher
	key_compaction_buf []byte
	disable_hashing    bool
	full_nodes_kept    uint16
}

func (self *hashing_ctx) reset() {
	self.enc_hash.Reset()
	self.enc_storage.Reset()
	self.hasher.Reset()
	self.disable_hashing = false
	self.full_nodes_kept = 0
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

func (self *hashing_ctx) hash_list_end(out *node_hash, list *rlp.ListHead, is_root bool) {
	if self.disable_hashing {
		return
	}
	self.enc_hash.ListEnd(list)
	if list.Size() < common.HashLength && !is_root {
		return
	}
	self.enc_hash.Flush(list, self.hasher.Write)
	*out = self.hasher.Hash()
	if !is_root {
		self.hasher.Reset()
		self.enc_hash.EraseSince(list)
		self.enc_hash.AppendString(*out)
	}
}

var ctx_pool = sync.Pool{New: func() interface{} {
	new_encoder := func() *rlp.Encoder {
		return rlp.NewEncoder(rlp.EncoderConfig{1 << 19, 1 << 8})
	}
	return &hashing_ctx{
		enc_hash:           new_encoder(),
		enc_storage:        new_encoder(),
		hasher:             util.NewHasher(),
		key_compaction_buf: make([]byte, common.HashLength*2),
	}
}}

// TODO parallel
// TODO cache full nodes
// TODO prettify
func (self *Trie) commit(n node, ctx *hashing_ctx) node {
	dot_draw_level(self.DotGraph, n)
	is_root := ctx == nil
	if is_root {
		if h := n.get_hash(); len(h) != 0 {
			return h
		}
		ctx = ctx_pool.Get().(*hashing_ctx)
		defer ctx_pool.Put(ctx)
		defer ctx.reset()
	}
	switch n := n.(type) {
	case node_hash:
		ctx.hash_append_string(n)
		ctx.enc_storage.AppendString(n)
		return n
	case *short_node:
		if len(n.hash) != 0 {
			ctx.hash_append_string(n.hash)
			ctx.toggle_hashing()
			defer ctx.toggle_hashing()
		}
		storage_list := ctx.enc_storage.ListStart()
		hash_list := ctx.hash_list_start()
		hashed_key_ext := hex_to_compact_(n.key_part, ctx.key_compaction_buf)
		ctx.enc_storage.AppendString(hashed_key_ext)
		ctx.hash_append_string(hashed_key_ext)
		val, has_val := n.val.(*value)
		if has_val {
			if !ctx.disable_hashing {
				ctx.hash_append_string(val.enc_hash)
			}
		} else {
			n.val = self.commit(n.val, ctx)
		}
		ctx.hash_list_end(&n.hash, hash_list, is_root)
		if has_val {
			if len(n.hash) != 0 {
				ctx.enc_storage.AppendString(n.hash)
			} else if len(val.enc_storage) <= self.schema.MaxStorageEncSizeToStoreInTrie() {
				ctx.enc_storage.AppendString(val.enc_storage)
			}
		}
		ctx.enc_storage.ListEnd(storage_list)
		if is_root {
			self.storage.PutAsync(COL_hash_to_node, n.hash, ctx.enc_storage.ToBytes(storage_list))
			return n.hash
		}
		return n
	case *full_node:
		if len(n.hash) != 0 {
			ctx.hash_append_string(n.hash)
			ctx.enc_storage.AppendString(n.hash)
			return n.hash
		}
		hash_list := ctx.hash_list_start()
		storage_list := ctx.enc_storage.ListStart()
		for i := 0; i < 16; i++ {
			if c := n.children[i]; c != nil {
				n.children[i] = self.commit(c, ctx)
			} else {
				ctx.hash_append_string(nil)
				ctx.enc_storage.AppendEmptyString()
			}
		}
		ctx.enc_storage.ListEnd(storage_list)
		ctx.hash_append_string(nil)
		ctx.hash_list_end(&n.hash, hash_list, is_root)
		if len(n.hash) != 0 {
			self.storage.PutAsync(COL_hash_to_node, n.hash, ctx.enc_storage.ToBytes(storage_list))
			if !is_root {
				ctx.enc_storage.EraseSince(storage_list)
				ctx.enc_storage.AppendString(n.hash)
			}
			return n.hash
		}
		return n
	}
	panic("impossible")
}

func (self *Trie) get_val_node_by_hex_k(hex_key []byte) *value {
	// TODO key buffer reuse
	enc_storage := self.get_val_enc_stoage_by_hashed_k(hexToKeybytes(hex_key))
	util.Assert(len(enc_storage) != 0)
	return &value{enc_storage, self.schema.StorageToHashEncoding(enc_storage)}
}

func (self *Trie) get_val_enc_stoage_by_hashed_k(hashed_key []byte) (enc_storage []byte) {
	return self.storage.GetCommitted(COL_flat_key_to_value, self.schema.FlatKey(hashed_key))
}

var nil_val_node *value = nil
