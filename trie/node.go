// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"io"
)

type node interface {
	fstring(string) string
	cached_hash() (hashNode, bool)
	canUnload(cachegen, cachelimit uint16) bool
}

type nodeFlag struct {
	hash  hashNode // cached hash of the node (may be nil)
	gen   uint16   // cache generation counter
	dirty bool     // whether the node has changes that must be written to the database
}

func (n *nodeFlag) canUnload(cachegen, cachelimit uint16) bool {
	return !n.dirty && cachegen-n.gen >= cachelimit
}

type fullNode struct {
	Children [17]node
	flags    nodeFlag
}

func (n *fullNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }
func (n *fullNode) cached_hash() (hashNode, bool)    { return n.flags.hash, n.flags.dirty }
func (n *fullNode) copy() *fullNode                  { copy := *n; return &copy }
func (n *fullNode) String() string                   { return n.fstring("") }

var indices = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f", "[17]"}

func (n *fullNode) fstring(ind string) string {
	resp := fmt.Sprintf("[\n%s  ", ind)
	for i, node := range &n.Children {
		if node == nil {
			resp += fmt.Sprintf("%s: <nil> ", indices[i])
		} else {
			resp += fmt.Sprintf("%s: %v", indices[i], node.fstring(ind+"  "))
		}
	}
	return resp + fmt.Sprintf("\n%s] ", ind)
}

var nilValueNode = valueNode(nil)

func (n *fullNode) EncodeRLP(w io.Writer) error {
	var nodes [17]node
	for i, child := range &n.Children {
		if child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

type shortNode struct {
	Key   []byte
	Val   node
	flags nodeFlag
}

func (n *shortNode) copy() *shortNode                 { copy := *n; return &copy }
func (n *shortNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }
func (n *shortNode) cached_hash() (hashNode, bool)    { return n.flags.hash, n.flags.dirty }
func (n *shortNode) String() string                   { return n.fstring("") }
func (n *shortNode) fstring(ind string) string {
	return fmt.Sprintf("{%x: %v} ", n.Key, n.Val.fstring(ind+"  "))
}

type hashNode []byte

func (n hashNode) canUnload(uint16, uint16) bool { return false }
func (n hashNode) cached_hash() (hashNode, bool) { return nil, true }
func (n hashNode) String() string                { return n.fstring("") }
func (n hashNode) fstring(string) string         { return fmt.Sprintf("<%x> ", []byte(n)) }

type valueNode []byte

func (n valueNode) canUnload(uint16, uint16) bool { return false }
func (n valueNode) cached_hash() (hashNode, bool) { return nil, true }
func (n valueNode) String() string                { return n.fstring("") }
func (n valueNode) fstring(string) string         { return fmt.Sprintf("%x ", []byte(n)) }

type value_node_resolver = func(key_extension, value []byte) valueNode

func mustDecodeNode(hash, buf []byte, cachegen uint16, value_node_resolver value_node_resolver) node {
	return decodeNode(hash, buf, cachegen, value_node_resolver)
}

func decodeNode(hash, buf []byte, cachegen uint16, value_node_resolver value_node_resolver) node {
	if len(buf) == 0 {
		panic(io.ErrUnexpectedEOF)
	}
	elems, _, err := rlp.SplitList(buf)
	util.PanicIfNotNil(err)
	switch c, _ := rlp.CountValues(elems); c {
	case 2:
		return decodeShort(hash, elems, cachegen, value_node_resolver)
	case 17:
		return decodeFull(hash, elems, cachegen, value_node_resolver)
	default:
		panic(fmt.Errorf("invalid number of list elements: %v", c))
	}
}

func decodeShort(hash, elems []byte, cachegen uint16, value_node_resolver value_node_resolver) node {
	kbuf, rest, err := rlp.SplitString(elems)
	util.PanicIfNotNil(err)
	flag := nodeFlag{hash: hash, gen: cachegen}
	key := compactToHex(kbuf)
	if hasTerm(key) {
		val, _, err := rlp.SplitString(rest)
		util.PanicIfNotNil(err)
		ret := &shortNode{Key: key, flags: flag}
		if len(val) > 0 {
			ret.Val = value_node_resolver(key, val)
		}
		return ret
	}
	r, _ := decodeRef(rest, cachegen, value_node_resolver)
	return &shortNode{key, r, flag}
}

func decodeFull(hash, elems []byte, cachegen uint16, value_node_resolver value_node_resolver) *fullNode {
	n := &fullNode{flags: nodeFlag{hash: hash, gen: cachegen}}
	for i := 0; i < 16; i++ {
		n.Children[i], elems = decodeRef(elems, cachegen, value_node_resolver)
	}
	val, _, err := rlp.SplitString(elems)
	util.PanicIfNotNil(err)
	if len(val) > 0 {
		n.Children[16] = value_node_resolver(nil, val)
	}
	return n
}

func decodeRef(buf []byte, cachegen uint16, value_node_resolver value_node_resolver) (node, []byte) {
	kind, val, rest, err := rlp.Split(buf)
	util.PanicIfNotNil(err)
	switch {
	case kind == rlp.List:
		// 'embedded' node reference. The encoding must be smaller
		// than a hash in order to be valid.
		if size := len(buf) - len(rest); size > common.HashLength {
			panic(fmt.Errorf("oversized embedded node (size is %d bytes, want size < %d)", size, common.HashLength))
		}
		return decodeNode(nil, buf, cachegen, value_node_resolver), rest
	case kind == rlp.String && len(val) == 0:
		// empty node
		return nil, rest
	case kind == rlp.String && len(val) == 32:
		return append(hashNode{}, val...), rest
	default:
		panic(fmt.Errorf("invalid RLP string size %d (want 0 or 32)", len(val)))
	}
}
