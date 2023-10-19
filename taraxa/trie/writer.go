package trie

import (
	"errors"
	"fmt"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type Writer struct {
	Reader
	root       node
	kbuf_0     hex_key
	kbuf_1     hex_key
	commit_ctx commit_context
	opts       WriterOpts
}
type WriterOpts struct {
	FullNodeLevelsToCache byte
}

func (self *Writer) Init(schema Schema, root_hash *common.Hash, opts WriterOpts) *Writer {
	asserts.Holds(opts.FullNodeLevelsToCache <= MaxDepth)
	self.Schema = schema
	if root_hash != nil {
		self.root = (*node_hash)(root_hash)
	}
	self.opts = opts
	return self
}

func (self *Writer) Commit(db_tx IO) *common.Hash {
	if self.root == nil {
		return nil
	}
	if h := self.root.get_hash(); h != nil {
		return h.common_hash()
	}
	self.commit_ctx.Reset()
	self.root = self.commit(db_tx, &self.commit_ctx, 0, self.kbuf_0[:0], self.root)
	return self.root.get_hash().common_hash()
}

// TODO parallel
func (self *Writer) commit(db_tx IO, ctx *commit_context, full_nodes_above byte, key_prefix []byte, n node) node {
	is_root := len(key_prefix) == 0
	switch n := n.(type) {
	case *node_hash:
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
					key := common.BytesToHash(hexToKeybytes(append(key_prefix, n.key_part...)))
					db_tx.PutValue(&key, val.enc_storage)
				}
				ctx.enc_hash.AppendString(val.enc_hash)
			} else {
				asserts.Holds(n.hash != nil)
			}
		} else {
			n.val = self.commit(db_tx, ctx, full_nodes_above, append(key_prefix, n.key_part...), n.val)
		}
		ctx.enc_hash.ListEnd(hash_list_start, is_root, &n.hash)
		if has_val {
			if n.hash != nil {
				ctx.enc_storage.AppendString(n.hash[:])
			} else if len(val.enc_storage) <= self.MaxValueSizeToStoreInTrie() {
				ctx.enc_storage.AppendString(val.enc_storage)
			}
		}
		ctx.enc_storage.ListEnd(storage_list_start)
		if is_root {
			fmt.Println("PutNode(is_root)", n.get_hash().common_hash().Hex(), common.Bytes2Hex(ctx.enc_storage.ToBytes(storage_list_start)))
			db_tx.PutNode(n.get_hash().common_hash(), ctx.enc_storage.ToBytes(storage_list_start))
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
				n.children[i] = self.commit(db_tx, ctx, full_nodes_above+1, append(key_prefix, i), child)
			} else {
				ctx.enc_hash.AppendString(nil)
				ctx.enc_storage.AppendString(nil)
			}
		}
		ctx.enc_storage.ListEnd(storage_list_start)
		ctx.enc_hash.ListEnd(hash_list_start, is_root, &n.hash)
		if n.hash != nil {
			fmt.Println("PutNode", n.get_hash().common_hash().Hex(), common.Bytes2Hex(ctx.enc_storage.ToBytes(storage_list_start)))
			db_tx.PutNode(n.get_hash().common_hash(), ctx.enc_storage.ToBytes(storage_list_start))
			if !is_root {
				ctx.enc_storage.RevertToListStart(storage_list_start)
				ctx.enc_storage.AppendString(n.hash[:])
			}
			if self.opts.FullNodeLevelsToCache <= full_nodes_above {
				return n.hash
			}
		}
		return n
	default:
		panic("impossible")
	}
}

func (self *Writer) Put(db_tx IO, k *common.Hash, v Value) {
	self.write(db_tx, k, value_node{v})
}

func (self *Writer) Delete(db_tx IO, k *common.Hash) {
	self.write(db_tx, k, nil_val_node)
}

func (self *Writer) write(db_tx IO, k *common.Hash, v value_node) {
	keybytes_to_hex(k[:], self.kbuf_0[:])
	if v != nil_val_node {
		self.root = self.mpt_insert(db_tx, self.root, 0, v)
		return
	}
	defer util.Recover(func(issue util.Any) {
		if issue != mpt_del_not_found {
			panic(issue)
		}
	})
	self.root = self.mpt_del(db_tx, self.root, 0)
	db_tx.PutValue(k, nil)
}

// TODO maybe dirty checking is worthwhile
func (self *Writer) mpt_insert(db_tx Input, n node, keypos int, value value_node) node {
	switch n := n.(type) {
	case *short_node:
		n.hash = nil
		matchlen := prefixLen(self.kbuf_0[keypos:], n.key_part)
		keypos_after_match := keypos + matchlen
		if keypos_after_match == HexKeyLen {
			n.val = value
			return n
		}
		if matchlen == len(n.key_part) {
			n.val = self.mpt_insert(db_tx, n.val, keypos_after_match, value)
			return n
		}
		junction := new(full_node)
		new_pivot := n.key_part[matchlen]
		junction.children[new_pivot] = self.shift(
			db_tx,
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
		n.children[self.kbuf_0[keypos]] = self.mpt_insert(db_tx, n.children[self.kbuf_0[keypos]], keypos+1, value)
		return n
	case *node_hash:
		return self.mpt_insert(db_tx, self.resolve(db_tx, n, self.kbuf_0[:keypos]), keypos, value)
	case nil:
		return &short_node{key_part: common.CopyBytes(self.kbuf_0[keypos:]), val: value}
	default:
	}
	panic("impossible")
}

// TODO maybe panic/recover harms performance
var mpt_del_not_found = errors.New("key not found")

func (self *Writer) mpt_del(db_tx Input, n node, keypos int) node {
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
		child := self.mpt_del(db_tx, n.val, keypos)
		if short_n, is := child.(*short_node); is {
			n.key_part, n.val = bin.Concat(n.key_part, short_n.key_part...), short_n.val
		} else {
			n.val = child
		}
		return n
	case *full_node:
		deletion_nibble := int8(self.kbuf_0[keypos])
		deletion_child := self.mpt_del(db_tx, n.children[deletion_nibble], keypos+1)
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
			only_child = self.resolve(db_tx, hash_n, self.kbuf_0[:keypos], only_child_nibble_b)
		}
		if short_n, is := only_child.(*short_node); is {
			short_n.hash = nil
			return self.shift(db_tx, short_n, self.kbuf_0[:keypos], only_child_nibble_b, short_n.key_part, true)
		}
		return &short_node{key_part: []byte{only_child_nibble_b}, val: only_child}
	case *node_hash:
		return self.mpt_del(db_tx, self.resolve(db_tx, n, self.kbuf_0[:keypos]), keypos)
	case nil:
		panic(mpt_del_not_found)
	default:
	}
	panic("impossible")
}

func (self *Writer) shift(db_tx Input, n *short_node, new_prefix []byte, pivot byte, new_suffix []byte, up bool) node {
	if n.val == nil_val_node {
		hex_key := append(append(append(self.kbuf_1[:0], new_prefix...), pivot), new_suffix...)
		n.val = self.resolve_val_n_by_hex_k(db_tx, hex_key)
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

func (self *Writer) resolve(db_tx Input, hash *node_hash, key_prefix_base []byte, key_prefix_rest ...byte) node {
	node, _ := self.Reader.resolve(db_tx, hash, append(append(self.kbuf_1[:0], key_prefix_base...), key_prefix_rest...))
	return node
}
