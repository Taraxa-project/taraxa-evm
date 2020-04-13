package trie

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type TrieWriter struct {
	db         DB
	root       node
	opts       TrieWriterOpts
	kbuf_0     hex_key
	kbuf_1     hex_key
	commit_ctx *commit_context
}
type TrieWriterOpts struct {
	FullNodeLevelsToCache byte
	AnticipatedDepth      byte
}

const MaxDepth = common.HashLength * 2
const hex_key_len = MaxDepth + 1
const hex_key_compact_len = common.HashLength + 1

type hex_key = [hex_key_len]byte
type hex_key_compact = [hex_key_compact_len]byte

func (self *TrieWriter) Init(db DB, root_hash *common.Hash, opts TrieWriterOpts) *TrieWriter {
	assert.Holds(opts.FullNodeLevelsToCache <= MaxDepth)
	assert.Holds(opts.AnticipatedDepth <= MaxDepth)
	self.db = db
	if root_hash != nil {
		self.root = (*node_hash)(root_hash)
	}
	self.opts = opts
	self.commit_ctx = get_commit_ctx(opts.AnticipatedDepth)
	return self
}

func (self *TrieWriter) Commit() *common.Hash {
	if self.root == nil {
		return nil
	}
	if h := self.root.get_hash(); h != nil {
		return h.common_hash()
	}
	defer self.commit_ctx.Reset()
	self.root = self.commit(self.commit_ctx, 0, self.kbuf_0[:0], self.root)
	return self.root.get_hash().common_hash()
}

// TODO parallel
func (self *TrieWriter) commit(ctx *commit_context, full_nodes_above byte, key_prefix []byte, n node) node {
	is_root := len(key_prefix) == 0
	switch n := n.(type) {
	case *node_hash:
		//assert.Holds(!ctx.enc_hash.disabled)
		ctx.enc_hash.AppendString(n[:])
		ctx.enc_storage.AppendString(n[:])
		return n
	case *short_node:
		if n.hash != nil {
			ctx.enc_hash.AppendString(n.hash[:])
			ctx.enc_hash.Toggle()
			defer ctx.enc_hash.Toggle()
		}
		storage_list_start, hash_list_start := ctx.enc_storage.ListStart(), ctx.enc_hash.ListStart()
		hashed_key_ext := hex_to_compact(n.key_part, &ctx.hex_key_compact_tmp)
		ctx.enc_storage.AppendString(hashed_key_ext)
		ctx.enc_hash.AppendString(hashed_key_ext)
		val_n, has_val := n.val.(value_node)
		var val internal_value
		if has_val {
			if val_n != nil_val_node {
				cached := false
				if val, cached = val_n.val.(internal_value); !cached {
					val.enc_storage, val.enc_hash = val_n.val.EncodeForTrie()
					val_n.val = val
					key := new(common.Hash)
					hex_to_keybytes(append(key_prefix, n.key_part...), key[:])
					self.db.PutValue(key, val.enc_storage)
				}
				ctx.enc_hash.AppendString(val.enc_hash)
			} else {
				assert.Holds(n.hash != nil)
			}
		} else {
			n.val = self.commit(ctx, full_nodes_above, append(key_prefix, n.key_part...), n.val)
		}
		ctx.enc_hash.ListEnd(hash_list_start, is_root, &n.hash)
		if has_val {
			if n.hash != nil {
				ctx.enc_storage.AppendString(n.hash[:])
			} else if len(val.enc_storage) <= self.db.MaxValueSizeToStoreInTrie() {
				ctx.enc_storage.AppendString(val.enc_storage)
			}
		}
		ctx.enc_storage.ListEnd(storage_list_start)
		if is_root {
			self.db.PutNode(n.hash.common_hash(), ctx.enc_storage.ToBytes(storage_list_start))
		}
		return n
	case *full_node:
		if n.hash != nil {
			ctx.enc_hash.AppendString(n.hash[:])
			ctx.enc_storage.AppendString(n.hash[:])
			if self.opts.FullNodeLevelsToCache <= full_nodes_above {
				return n.hash
			}
			return n
		}
		hash_list_start, storage_list_start := ctx.enc_hash.ListStart(), ctx.enc_storage.ListStart()
		for i := byte(0); i < full_node_child_cnt; i++ {
			if child := n.children[i]; child != nil {
				n.children[i] = self.commit(ctx, full_nodes_above+1, append(key_prefix, i), child)
			} else {
				ctx.enc_hash.AppendString(nil)
				ctx.enc_storage.AppendString(nil)
			}
		}
		ctx.enc_storage.ListEnd(storage_list_start)
		ctx.enc_hash.AppendString(nil)
		ctx.enc_hash.ListEnd(hash_list_start, is_root, &n.hash)
		if n.hash != nil {
			self.db.PutNode(n.hash.common_hash(), ctx.enc_storage.ToBytes(storage_list_start))
			if !is_root {
				ctx.enc_storage.RevertToListStart(storage_list_start)
				ctx.enc_storage.AppendString(n.hash[:])
			}
			if self.opts.FullNodeLevelsToCache <= full_nodes_above {
				return n.hash
			}
		}
		return n
	}
	panic("impossible")
}

type Value interface {
	EncodeForTrie() (enc_storage, enc_hash []byte)
}

func (self *TrieWriter) Put(k *common.Hash, v Value) {
	self.write(k, value_node{v})
}

func (self *TrieWriter) Delete(k *common.Hash) {
	self.write(k, nil_val_node)
}

func (self *TrieWriter) write(k *common.Hash, v value_node) {
	keybytes_to_hex(k[:], self.kbuf_0[:])
	if v != nil_val_node {
		self.root = self.mpt_insert(self.root, 0, v)
		return
	}
	defer util.Recover(func(issue util.Any) {
		if issue != mpt_del_not_found {
			panic(issue)
		}
	})
	self.root = self.mpt_del(self.root, 0)
	self.db.DeleteValue(k)
}

// TODO maybe dirty checking is worthwhile
func (self *TrieWriter) mpt_insert(n node, keypos int, value value_node) node {
	switch n := n.(type) {
	case *short_node:
		n.hash = nil
		matchlen := prefixLen(self.kbuf_0[keypos:], n.key_part)
		keypos_after_match := keypos + matchlen
		if keypos_after_match == hex_key_len {
			n.val = value
			return n
		}
		if matchlen == len(n.key_part) {
			n.val = self.mpt_insert(n.val, keypos_after_match, value)
			return n
		}
		junction := new(full_node)
		new_pivot := n.key_part[matchlen]
		junction.children[new_pivot] = self.shift(
			n,
			self.kbuf_0[:keypos_after_match],
			new_pivot,
			n.key_part[matchlen+1:],
			false,
		)
		junction.children[self.kbuf_0[keypos_after_match]] = &short_node{
			key_part: common.CopyBytes(self.kbuf_0[keypos_after_match+1:]),
			val:      value,
		}
		if matchlen == 0 {
			return junction
		}
		return &short_node{key_part: common.CopyBytes(self.kbuf_0[keypos:keypos_after_match]), val: junction}
	case *full_node:
		n.hash = nil
		n.children[self.kbuf_0[keypos]] = self.mpt_insert(n.children[self.kbuf_0[keypos]], keypos+1, value)
		return n
	case *node_hash:
		return self.mpt_insert(self.resolve(n, self.kbuf_0[:keypos]), keypos, value)
	case nil:
		return &short_node{key_part: common.CopyBytes(self.kbuf_0[keypos:]), val: value}
	}
	panic("impossible")
}

// TODO maybe panic/recover harms performance
var mpt_del_not_found = errors.New("key not found")

func (self *TrieWriter) mpt_del(n node, keypos int) node {
	switch n := n.(type) {
	case *short_node:
		matchlen := prefixLen(self.kbuf_0[keypos:], n.key_part)
		if matchlen != len(n.key_part) {
			panic(mpt_del_not_found)
		}
		keypos += matchlen
		if keypos == len(self.kbuf_0) {
			return nil
		}
		n.hash = nil
		child := self.mpt_del(n.val, keypos)
		if short_n, is := child.(*short_node); is {
			n.key_part, n.val = bin.Concat(n.key_part, short_n.key_part...), short_n.val
		} else {
			n.val = child
		}
		return n
	case *full_node:
		deletion_nibble := int8(self.kbuf_0[keypos])
		deletion_child := self.mpt_del(n.children[deletion_nibble], keypos+1)
		only_child_nibble := int8(-1)
		if deletion_child == nil {
			for i := int8(0); i < full_node_child_cnt; i++ {
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
			only_child = self.resolve(hash_n, self.kbuf_0[:keypos], only_child_nibble_b)
		}
		if short_n, is := only_child.(*short_node); is {
			short_n.hash = nil
			return self.shift(short_n, self.kbuf_0[:keypos], only_child_nibble_b, short_n.key_part, true)
		}
		return &short_node{key_part: []byte{only_child_nibble_b}, val: only_child}
	case *node_hash:
		return self.mpt_del(self.resolve(n, self.kbuf_0[:keypos]), keypos)
	case nil:
		panic(mpt_del_not_found)
	}
	panic("impossible")
}

func (self *TrieWriter) shift(n *short_node, new_prefix []byte, pivot byte, new_suffix []byte, up bool) node {
	// TODO reuse buffers
	if n.val == nil_val_node {
		hex_key := append(append(append(self.kbuf_1[:0], new_prefix...), pivot), new_suffix...)
		n.val = self.get_val_node_by_hex_k(hex_key)
		if up {
			n.key_part = common.CopyBytes(hex_key[len(new_prefix):])
			return n
		}
	}
	if up {
		n.key_part = append(append(make([]byte, 0, 1+len(new_suffix)), pivot), new_suffix...)
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
	enc := self.db.GetNode(hash.common_hash())
	assert.Holds(len(enc) != 0)
	key_prefix := append(append(self.kbuf_1[:0], key_prefix_base...), key_prefix_rest...)
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
			assert.Holds(0 < l && l <= self.db.MaxValueSizeToStoreInTrie())
			ret.val = value_node{internal_value{content, self.db.ValueStorageToHashEncoding(content)}}
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
	for i := byte(0); i < full_node_child_cnt; i++ {
		ret.children[i], enc = self.dec_node(append(key_prefix, i), nil, enc)
	}
	return ret
}

func (self *TrieWriter) get_val_node_by_hex_k(hex_key []byte) value_node {
	var key common.Hash
	hex_to_keybytes(hex_key, key[:])
	enc_storage := self.db.GetValue(&key)
	assert.Holds(len(enc_storage) != 0)
	return value_node{internal_value{enc_storage, self.db.ValueStorageToHashEncoding(enc_storage)}}
}

var nil_val_node = value_node{nil}
