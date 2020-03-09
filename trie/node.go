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
	cached_hash() (hashNode, bool)
	canUnload(cachegen, cachelimit uint16) bool
}

type node_enc_strategy interface {
	enc_full(n *fullNode, w io.Writer) error
	enc_short(n *shortNode, w io.Writer) error
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

func (n *fullNode) EncodeRLP(w io.Writer) error {
	return w.(rlp.Parameterized).
		Params()[0].(node_enc_strategy).
		enc_full(n, w)
}

type shortNode struct {
	Key   []byte
	Val   node
	flags nodeFlag
}

func (n *shortNode) copy() *shortNode                 { copy := *n; return &copy }
func (n *shortNode) canUnload(gen, limit uint16) bool { return n.flags.canUnload(gen, limit) }
func (n *shortNode) cached_hash() (hashNode, bool)    { return n.flags.hash, n.flags.dirty }

func (n *shortNode) EncodeRLP(w io.Writer) error {
	return w.(rlp.Parameterized).
		Params()[0].(node_enc_strategy).
		enc_short(n, w)
}

type hashNode []byte

func (n hashNode) canUnload(uint16, uint16) bool { return false }
func (n hashNode) cached_hash() (hashNode, bool) { return nil, true }

type valueNode []byte

func (n valueNode) canUnload(uint16, uint16) bool { return false }
func (n valueNode) cached_hash() (hashNode, bool) { return nil, true }

type value_resolver = func(mpt_key_hex []byte) valueNode

var nilValueNode = valueNode(nil)

func decodeNode(path, hash, buf []byte, cachegen uint16, value_resolver value_resolver) node {
	if len(buf) == 0 {
		panic(io.ErrUnexpectedEOF)
	}
	elems, _, err := rlp.SplitList(buf)
	util.PanicIfNotNil(err)
	switch c, _ := rlp.CountValues(elems); c {
	case 1, 2:
		return decodeShort(path, hash, elems, cachegen, value_resolver)
	case 16:
		return decodeFull(path, hash, elems, cachegen, value_resolver)
	default:
		panic(fmt.Errorf("invalid number of list elements: %v", c))
	}
}

func decodeShort(path, hash, elems []byte, cachegen uint16, value_resolver value_resolver) node {
	kbuf, rest, err := rlp.SplitString(elems)
	util.PanicIfNotNil(err)
	flag := nodeFlag{hash: hash, gen: cachegen}
	key := compactToHex(kbuf)
	path = append(path, key...)
	if hasTerm(key) {
		return &shortNode{key, value_resolver(path), flag}
	}
	forward_hash, _ := decodeRef(path, rest, cachegen, value_resolver)
	return &shortNode{key, forward_hash, flag}
}

func decodeFull(path, hash, elems []byte, cachegen uint16, value_resolver value_resolver) *fullNode {
	n := &fullNode{flags: nodeFlag{hash: hash, gen: cachegen}}
	for i := byte(0); i < 16; i++ {
		n.Children[i], elems = decodeRef(append(path, i), elems, cachegen, value_resolver)
	}
	if hasTerm(path) {
		n.Children[16] = value_resolver(path)
	}
	return n
}

func decodeRef(path, buf []byte, cachegen uint16, value_resolver value_resolver) (node, []byte) {
	kind, val, rest, err := rlp.Split(buf)
	util.PanicIfNotNil(err)
	switch {
	case kind == rlp.List:
		// 'embedded' node reference. The encoding must be smaller
		// than a hash in order to be valid.
		if size := len(buf) - len(rest); size > common.HashLength {
			panic(fmt.Errorf("oversized embedded node (size is %d bytes, want size < %d)", size, common.HashLength))
		}
		return decodeNode(path, nil, buf, cachegen, value_resolver), rest
	case kind == rlp.String && len(val) == 0:
		// empty node
		return nil, rest
	case kind == rlp.String && len(val) == common.HashLength:
		return append(hashNode{}, val...), rest
	default:
		panic(fmt.Errorf("invalid RLP string size %d (want 0 or 32)", len(val)))
	}
}
