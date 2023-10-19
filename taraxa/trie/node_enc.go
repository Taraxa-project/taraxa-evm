// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package trie

import (
	"github.com/ethereum/go-ethereum/rlp"
)

func nodeToBytes(n node, res *Resolver) []byte {
	w := rlp.NewEncoderBuffer(nil)
	n.encode(w, res)
	result := w.ToBytes()
	w.Flush()
	return result
}

func (n *full_node) encode(w rlp.EncoderBuffer, res *Resolver) {
	offset := w.List()
	for i, c := range n.children {
		if c != nil && c != nil_val_node {
			var cr *Resolver
			if res != nil {
				cr = res.CopyWithPrefix(byte(i))
			}
			c.get_hash().encode(w, cr)
		} else {
			w.Write(rlp.EmptyString)
		}
	}
	w.ListEnd(offset)
}

func (n *short_node) encode(w rlp.EncoderBuffer, res *Resolver) {
	offset := w.List()
	w.WriteBytes(n.key_part)
	if n.val == nil {
		if res != nil {
			n.val = res.resolve_val_n(n.key_part)
		}
	}
	if n.val != nil {
		_, eh := n.val.(value_node).val.EncodeForTrie()
		w.WriteBytes(eh)
	} else {
		w.Write(rlp.EmptyString)
	}
	w.ListEnd(offset)
}

func (n node_hash) encode(w rlp.EncoderBuffer, res *Resolver) {
	w.WriteBytes(n.common_hash().Bytes())
}

func (n value_node) encode(w rlp.EncoderBuffer, res *Resolver) {

}
