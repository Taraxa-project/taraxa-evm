// Copyright 2016 The go-ethereum Authors
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
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/emicklei/dot"
	"hash"
	"math/rand"
	"reflect"
	"strconv"
	"sync"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"golang.org/x/crypto/sha3"
)

type hasher struct {
	tmp        sliceBuffer
	sha        keccakState
	cachegen   uint16
	cachelimit uint16
	dot_g      *dot.Graph
}
type hasher_store_strategy = func(hash hashNode, n node, n_enc []byte) error

// keccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type keccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

type sliceBuffer []byte

func (b *sliceBuffer) Write(data []byte) (n int, err error) {
	*b = append(*b, data...)
	return len(data), nil
}

func (b *sliceBuffer) Reset() {
	*b = (*b)[:0]
}

// hashers live in a global db.
var hasherPool = sync.Pool{
	New: func() interface{} {
		return &hasher{
			tmp: make(sliceBuffer, 0, 550), // cap is as large as a full fullNode.
			sha: sha3.NewLegacyKeccak256().(keccakState),
		}
	},
}

func newHasher(cachegen, cachelimit uint16) *hasher {
	h := hasherPool.Get().(*hasher)
	h.cachegen, h.cachelimit = cachegen, cachelimit
	return h
}

func returnHasherToPool(h *hasher) {
	h.dot_g = nil
	hasherPool.Put(h)
}

// hash collapses a node down into a hash node, also returning a copy of the
// original node initialized with the computed hash to replace the original one.
func (h *hasher) hash(n node, force bool, store hasher_store_strategy) (node, node, error) {
	// If we're not storing the node, just hashing, use available cached data
	if hash, dirty := n.cached_hash(); hash != nil {
		if store == nil {
			return hash, n, nil
		}
		if n.canUnload(h.cachegen, h.cachelimit) {
			// Unload the node from cache. All of its subnodes will have a lower or equal
			// cache generation number.
			cacheUnloadCounter.Inc(1)
			return hash, hash, nil
		}
		if !dirty {
			return hash, n, nil
		}
	}
	// Trie not processed yet or needs storage, walk the children
	collapsed, cached, err := h.hashChildren(n, store)
	if err != nil {
		return hashNode{}, n, err
	}
	hashed, err := h.hash_and_maybe_store(collapsed, force, store)
	if err != nil {
		return hashNode{}, n, err
	}
	// Cache the hash of the node for later reuse and remove
	// the dirty flag in commit mode. It's fine to assign these values directly
	// without copying the node first because hashChildren copies it.
	cachedHash, _ := hashed.(hashNode)
	switch cn := cached.(type) {
	case *shortNode:
		cn.flags.hash = cachedHash
		if store != nil {
			cn.flags.dirty = false
		}
	case *fullNode:
		cn.flags.hash = cachedHash
		if store != nil {
			cn.flags.dirty = false
		}
	}
	return hashed, cached, nil
}

// hashChildren replaces the children of a node with their hashes if the encoded
// size of the child is larger than a hash, returning the collapsed node as well
// as a replacement for the original node with the child hashes cached in.
func (h *hasher) hashChildren(original node, store hasher_store_strategy) (node, node, error) {
	var err error
	switch n := original.(type) {
	case *shortNode:
		h.dot_edge(n, n.Val)
		// Hash the short node's child, caching the newly hashed subtree
		collapsed, cached := n.copy(), n.copy()
		collapsed.Key = hexToCompact(n.Key)
		cached.Key = common.CopyBytes(n.Key)
		if _, ok := n.Val.(valueNode); !ok {
			collapsed.Val, cached.Val, err = h.hash(n.Val, false, store)
			if err != nil {
				return original, original, err
			}
		}
		return collapsed, cached, nil
	case *fullNode:
		for _, c := range n.Children {
			h.dot_edge(n, c)
		}
		// Hash the full node's children, caching the newly hashed subtrees
		collapsed, cached := n.copy(), n.copy()
		for i := 0; i < 16; i++ {
			if n.Children[i] != nil {
				collapsed.Children[i], cached.Children[i], err = h.hash(n.Children[i], false, store)
				if err != nil {
					return original, original, err
				}
			}
		}
		cached.Children[16] = n.Children[16]
		if nn := n.Children[16]; nn != nil {
			_ = nn.(valueNode)
		}
		return collapsed, cached, nil
	case hashNode:
		return n, original, nil
	default:
		panic("impossible")
	}
}

func (h *hasher) hash_and_maybe_store(n node, force bool, store hasher_store_strategy) (node, error) {
	// Don't store hashes or empty nodes.
	util.Assert(n != nil, "impossible")
	if _, ok := n.(valueNode); ok {
		panic("impossible")
	}
	if _, isHash := n.(hashNode); isHash {
		return n, nil
	}
	// Generate the RLP encoding of the node
	h.tmp.Reset()
	if err := rlp.Encode(&h.tmp, n); err != nil {
		panic("encode error: " + err.Error())
	}
	if len(h.tmp) < 32 && !force {
		return n, nil // Nodes smaller than 32 bytes are stored inside their parent
	}
	// Larger nodes are replaced by their hash and stored in the database.
	hash, _ := n.cached_hash()
	if hash == nil {
		hash = h.makeHashNode(h.tmp)
	}
	if store != nil {
		if err := store(hash, n, h.tmp); err != nil {
			return nil, err
		}
	}
	return hash, nil
}

func (h *hasher) makeHashNode(data []byte) hashNode {
	n := make(hashNode, h.sha.Size())
	h.sha.Reset()
	h.sha.Write(data)
	h.sha.Read(n)
	return n
}

func (self *hasher) dot_node(n node) (ret dot.Node) {
	switch n := n.(type) {
	case *shortNode, *fullNode:
		reflect_n := reflect.ValueOf(n)
		ret = self.dot_g.Node(fmt.Sprint(reflect_n.Pointer()))
		ret.Label(reflect_n.Type().String())
	default:
		ret = self.dot_g.Node(strconv.FormatUint(rand.Uint64(), 10))
		if n == nil {
			ret.Label("NULL")
		} else {
			ret.Label(reflect.ValueOf(n).Type().String())
			if _, ok := n.(valueNode); ok {
				self.dot_g.AddToSameRank("leaves", ret)
			}
		}
	}
	return
}

func (self *hasher) dot_edge(from, to node) {
	if self.dot_g == nil {
		return
	}
	self.dot_g.Edge(self.dot_node(from), self.dot_node(to))
}
