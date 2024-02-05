package trie

import (
	"errors"

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

func (w *Writer) Init(schema Schema, root_hash *common.Hash, opts WriterOpts) *Writer {
	asserts.Holds(opts.FullNodeLevelsToCache <= MaxDepth)
	w.Schema = schema
	if root_hash != nil {
		w.root = (*node_hash)(root_hash)
	}
	w.opts = opts
	return w
}

func (w *Writer) Commit(db_tx IO) *common.Hash {
	if w.root == nil {
		return nil
	}
	if h := w.root.get_hash(); h != nil {
		return h.common_hash()
	}
	w.commit_ctx.Reset()
	w.root = w.commit(db_tx, &w.commit_ctx, 0, w.kbuf_0[:0], w.root)
	return w.root.get_hash().common_hash()
}

// TODO parallel
func (w *Writer) commit(db_tx IO, ctx *commit_context, full_nodes_above byte, key_prefix []byte, n node) node {
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
			n.val = w.commit(db_tx, ctx, full_nodes_above, append(key_prefix, n.key_part...), n.val)
		}
		ctx.enc_hash.ListEnd(hash_list_start, is_root, &n.hash)
		if has_val {
			if n.hash != nil {
				ctx.enc_storage.AppendString(n.hash[:])
			} else if len(val.enc_storage) <= w.MaxValueSizeToStoreInTrie() {
				ctx.enc_storage.AppendString(val.enc_storage)
			}
		}
		ctx.enc_storage.ListEnd(storage_list_start)
		if is_root {
			db_tx.PutNode(n.get_hash().common_hash(), ctx.enc_storage.ToBytes(storage_list_start))
		}
		return n
	case *full_node:
		if n.hash != nil {
			ctx.enc_hash.AppendString(n.hash[:])
			ctx.enc_storage.AppendString(n.hash[:])
			if w.opts.FullNodeLevelsToCache <= full_nodes_above {
				return n.hash
			}
			return n
		}
		hash_list_start, storage_list_start := ctx.enc_hash.ListStart(), ctx.enc_storage.ListStart()
		for i := byte(0); i < full_node_child_cnt; i++ {
			if child := n.children[i]; child != nil {
				n.children[i] = w.commit(db_tx, ctx, full_nodes_above+1, append(key_prefix, i), child)
			} else {
				ctx.enc_hash.AppendString(nil)
				ctx.enc_storage.AppendString(nil)
			}
		}
		ctx.enc_storage.ListEnd(storage_list_start)
		ctx.enc_hash.AppendString(nil)
		ctx.enc_hash.ListEnd(hash_list_start, is_root, &n.hash)
		if n.hash != nil {
			db_tx.PutNode(n.get_hash().common_hash(), ctx.enc_storage.ToBytes(storage_list_start))
			if !is_root {
				ctx.enc_storage.RevertToListStart(storage_list_start)
				ctx.enc_storage.AppendString(n.hash[:])
			}
			if w.opts.FullNodeLevelsToCache <= full_nodes_above {
				return n.hash
			}
		}
		return n
	default:
		panic("impossible")
	}
}

func (w *Writer) Put(db_tx IO, k *common.Hash, v Value) {
	w.write(db_tx, k, value_node{v})
}

func (w *Writer) Delete(db_tx IO, k *common.Hash) {
	w.write(db_tx, k, nil_val_node)
}

func (w *Writer) write(db_tx IO, k *common.Hash, v value_node) {
	keybytes_to_hex(k[:], w.kbuf_0[:])
	if v != nil_val_node {
		w.root = w.mpt_insert(db_tx, w.root, 0, v)
		return
	}
	defer util.Recover(func(issue util.Any) {
		if issue != err_mpt_del_not_found {
			panic(issue)
		}
	})
	w.root = w.mpt_del(db_tx, w.root, 0)
	db_tx.PutValue(k, nil)
}

// TODO maybe dirty checking is worthwhile
func (w *Writer) mpt_insert(db_tx Input, n node, keypos int, value value_node) node {
	switch n := n.(type) {
	case *short_node:
		n.hash = nil
		matchlen := prefixLen(w.kbuf_0[keypos:], n.key_part)
		keypos_after_match := keypos + matchlen
		if keypos_after_match == HexKeyLen {
			n.val = value
			return n
		}
		if matchlen == len(n.key_part) {
			n.val = w.mpt_insert(db_tx, n.val, keypos_after_match, value)
			return n
		}
		junction := new(full_node)
		new_pivot := n.key_part[matchlen]
		junction.children[new_pivot] = w.shift(
			db_tx,
			n,
			w.kbuf_0[:keypos_after_match],
			new_pivot,
			n.key_part[matchlen+1:],
			false,
		)
		junction.children[w.kbuf_0[keypos_after_match]] = &short_node{
			key_part: common.CopyBytes(w.kbuf_0[keypos_after_match+1:]),
			val:      value,
		}
		if matchlen == 0 {
			return junction
		}
		return &short_node{key_part: common.CopyBytes(w.kbuf_0[keypos:keypos_after_match]), val: junction}
	case *full_node:
		n.hash = nil
		n.children[w.kbuf_0[keypos]] = w.mpt_insert(db_tx, n.children[w.kbuf_0[keypos]], keypos+1, value)
		return n
	case *node_hash:
		return w.mpt_insert(db_tx, w.resolve(db_tx, n, w.kbuf_0[:keypos]), keypos, value)
	case nil:
		return &short_node{key_part: common.CopyBytes(w.kbuf_0[keypos:]), val: value}
	default:
	}
	panic("impossible")
}

// TODO maybe panic/recover harms performance
var err_mpt_del_not_found = errors.New("key not found")

func (w *Writer) mpt_del(db_tx Input, n node, keypos int) node {
	switch n := n.(type) {
	case *short_node:
		matchlen := prefixLen(w.kbuf_0[keypos:], n.key_part)
		if matchlen != len(n.key_part) {
			panic(err_mpt_del_not_found)
		}
		keypos += matchlen
		if keypos == len(w.kbuf_0) {
			return nil
		}
		n.hash = nil
		child := w.mpt_del(db_tx, n.val, keypos)
		if short_n, is := child.(*short_node); is {
			n.key_part, n.val = bin.Concat(n.key_part, short_n.key_part...), short_n.val
		} else {
			n.val = child
		}
		return n
	case *full_node:
		deletion_nibble := int8(w.kbuf_0[keypos])
		deletion_child := w.mpt_del(db_tx, n.children[deletion_nibble], keypos+1)
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
			only_child = w.resolve(db_tx, hash_n, w.kbuf_0[:keypos], only_child_nibble_b)
		}
		if short_n, is := only_child.(*short_node); is {
			short_n.hash = nil
			return w.shift(db_tx, short_n, w.kbuf_0[:keypos], only_child_nibble_b, short_n.key_part, true)
		}
		return &short_node{key_part: []byte{only_child_nibble_b}, val: only_child}
	case *node_hash:
		return w.mpt_del(db_tx, w.resolve(db_tx, n, w.kbuf_0[:keypos]), keypos)
	case nil:
		panic(err_mpt_del_not_found)
	default:
	}
	panic("impossible")
}

func (w *Writer) shift(db_tx Input, n *short_node, new_prefix []byte, pivot byte, new_suffix []byte, up bool) node {
	if n.val == nil_val_node {
		hex_key := append(append(append(w.kbuf_1[:0], new_prefix...), pivot), new_suffix...)
		n.val = w.resolve_val_n_by_hex_k(db_tx, hex_key)
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

func (w *Writer) resolve(db_tx Input, hash *node_hash, key_prefix_base []byte, key_prefix_rest ...byte) node {
	node, _ := w.Reader.resolve(db_tx, hash, append(append(w.kbuf_1[:0], key_prefix_base...), key_prefix_rest...))
	return node
}
