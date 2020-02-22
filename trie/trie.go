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

// Package trie implements Merkle Patricia Tries.
package trie

import (
	"bytes"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/metrics"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/emicklei/dot"
)

var (
	zeroHash           = make([]byte, common.HashLength)
	emptyRoot          = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	emptyState         = crypto.Keccak256Hash(nil)
	cacheMissCounter   = metrics.NewRegisteredCounter("trie/cachemiss", nil)
	cacheUnloadCounter = metrics.NewRegisteredCounter("trie/cacheunload", nil)
)

func CacheMisses() int64 {
	return cacheMissCounter.Count()
}

func CacheUnloads() int64 {
	return cacheUnloadCounter.Count()
}

type Trie struct {
	db                   Database
	root                 node
	cachegen, cachelimit uint16
	Dot_g                *dot.Graph
	storage_strat        StorageStrategy
}
type StorageStrategy = interface {
	MapKey(key []byte) (mpt_key, flat_key []byte, err error)
}

func New(root *common.Hash, db Database, cachelimit uint16, storage_strat StorageStrategy) (*Trie, error) {
	util.Assert(db != nil)
	if storage_strat == nil {
		storage_strat = DefaultStorageStrategy(0)
	}
	trie := &Trie{
		db:            db,
		cachelimit:    cachelimit,
		storage_strat: storage_strat,
	}
	if root_b := root[:]; bytes.Compare(root_b, zeroHash) != 0 && bytes.Compare(root_b, emptyRoot[:]) != 0 {
		rootnode, err := trie.resolve(root_b, nil)
		if err != nil {
			return nil, err
		}
		trie.root = rootnode
	}
	return trie, nil
}

func NewSecure(root *common.Hash, db Database, cachelimit uint16) (*Trie, error) {
	return New(root, db, cachelimit, KeyHashingStorageStrategy(0))
}

func (self *Trie) NodeIterator(start []byte) NodeIterator {
	return newNodeIterator(self, start)
}

func (self *Trie) Get(key []byte) ([]byte, error) {
	mpt_key, _, err_0 := self.storage_strat.MapKey(key)
	if err_0 != nil {
		return nil, err_0
	}
	mpt_key_hex := keybytesToHex(mpt_key)
	value, newroot, didResolve, err_1 := self.mpt_get(self.root, mpt_key_hex, 0)
	if err_1 == nil && didResolve {
		self.root = newroot
	}
	//v, _ := t.db.Get(flat_key)
	//util.Assert(bytes.Compare(v, value) == 0)
	return value, err_1
}

func (self *Trie) Insert(key, value []byte) error {
	mpt_key, _, err := self.storage_strat.MapKey(key)
	if err != nil {
		return err
	}
	mpt_key_hex := keybytesToHex(mpt_key)
	//t.db.GetTransaction().Put(flat_key, value)
	if len(value) != 0 {
		_, n, err := self.mpt_insert(self.root, nil, mpt_key_hex, valueNode(value))
		if err != nil {
			return err
		}
		self.root = n
	} else {
		_, n, err := self.mpt_del(self.root, nil, mpt_key_hex)
		if err != nil {
			return err
		}
		self.root = n
	}
	return nil
}

func (self *Trie) Delete(key []byte) error {
	return self.Insert(key, nil)
}

func (self *Trie) Hash() (ret common.Hash) {
	ret, self.root, _ = self.hashRoot(nil)
	return
}

func (self *Trie) Commit() (ret common.Hash, err error) {
	ret, self.root, err = self.hashRoot(self.store)
	self.cachegen++
	return
}

func (self *Trie) mpt_get(origNode node, key_hex []byte, pos int) (value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, false, nil
	case valueNode:
		return n, n, false, nil
	case *shortNode:
		if len(key_hex)-pos < len(n.Key) || !bytes.Equal(n.Key, key_hex[pos:pos+len(n.Key)]) {
			// key not found in trie
			return nil, n, false, nil
		}
		value, newnode, didResolve, err = self.mpt_get(n.Val, key_hex, pos+len(n.Key))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
			n.flags.gen = self.cachegen
		}
		return value, n, didResolve, err
	case *fullNode:
		value, newnode, didResolve, err = self.mpt_get(n.Children[key_hex[pos]], key_hex, pos+1)
		if err == nil && didResolve {
			n = n.copy()
			n.flags.gen = self.cachegen
			n.Children[key_hex[pos]] = newnode
		}
		return value, n, didResolve, err
	case hashNode:
		child, err := self.resolve(n, key_hex[:pos])
		if err != nil {
			return nil, n, true, err
		}
		value, newnode, _, err := self.mpt_get(child, key_hex, pos)
		return value, newnode, true, err
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", origNode, origNode))
	}
}

func (self *Trie) mpt_insert(n node, key_hex_prefix, key_hex_rest []byte, value node) (bool, node, error) {
	if len(key_hex_rest) == 0 {
		if v, ok := n.(valueNode); ok {
			return !bytes.Equal(v, value.(valueNode)), value, nil
		}
		return true, value, nil
	}
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key_hex_rest, n.Key)
		// If the whole key matches, keep this short node as is
		// and only update the value.
		if matchlen == len(n.Key) {
			dirty, nn, err := self.mpt_insert(n.Val, append(key_hex_prefix, key_hex_rest[:matchlen]...), key_hex_rest[matchlen:], value, )
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, self.newFlag()}, nil
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: self.newFlag()}
		var err error
		_, branch.Children[n.Key[matchlen]], err = self.mpt_insert(nil, append(key_hex_prefix, n.Key[:matchlen+1]...), n.Key[matchlen+1:], n.Val, )
		if err != nil {
			return false, nil, err
		}
		_, branch.Children[key_hex_rest[matchlen]], err = self.mpt_insert(nil, append(key_hex_prefix, key_hex_rest[:matchlen+1]...), key_hex_rest[matchlen+1:], value, )
		if err != nil {
			return false, nil, err
		}
		// Replace this shortNode with the branch if it occurs at index 0.
		if matchlen == 0 {
			return true, branch, nil
		}
		// Otherwise, replace it with a short node leading up to the branch.
		return true, &shortNode{key_hex_rest[:matchlen], branch, self.newFlag()}, nil
	case *fullNode:
		dirty, nn, err := self.mpt_insert(n.Children[key_hex_rest[0]], append(key_hex_prefix, key_hex_rest[0]), key_hex_rest[1:], value, )
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = self.newFlag()
		n.Children[key_hex_rest[0]] = nn
		return true, n, nil
	case nil:
		return true, &shortNode{key_hex_rest, value, self.newFlag()}, nil
	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and insert into it. This leaves all child nodes on
		// the path to the value in the trie.
		panic("Not yet")
		rn, err := self.resolve(n, key_hex_prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := self.mpt_insert(rn, key_hex_prefix, key_hex_rest, value)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}
}

func (self *Trie) mpt_del(n node, key_hex_prefix, key_hex_rest []byte) (bool, node, error) {
	switch n := n.(type) {
	case *shortNode:
		matchlen := prefixLen(key_hex_rest, n.Key)
		if matchlen < len(n.Key) {
			return false, n, nil // don't replace n on mismatch
		}
		if matchlen == len(key_hex_rest) {
			return true, nil, nil // remove n entirely for whole matches
		}
		// The key is longer than n.Key. Remove the remaining suffix
		// from the subtrie. Child can never be nil here since the
		// subtrie must contain at least two other values with keys
		// longer than n.Key.
		dirty, child, err := self.mpt_del(n.Val, append(key_hex_prefix, key_hex_rest[:len(n.Key)]...), key_hex_rest[len(n.Key):], )
		if !dirty || err != nil {
			return false, n, err
		}
		switch child := child.(type) {
		case *shortNode:
			// Deleting from the subtrie reduced it to another
			// short node. Merge the nodes to avoid creating a
			// shortNode{..., shortNode{...}}. Use concat (which
			// always creates a new slice) instead of append to
			// avoid modifying n.Key since it might be shared with
			// other nodes.
			return true, &shortNode{concat(n.Key, child.Key...), child.Val, self.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, self.newFlag()}, nil
		}
	case *fullNode:
		key_hex_prefix = append(key_hex_prefix, key_hex_rest[0])
		dirty, nn, err := self.mpt_del(n.Children[key_hex_rest[0]], key_hex_prefix, key_hex_rest[1:], )
		if !dirty || err != nil {
			return false, n, err
		}
		n = n.copy()
		n.flags = self.newFlag()
		n.Children[key_hex_rest[0]] = nn
		// Check how many non-nil entries are left after deleting and
		// reduce the full node to a short node if only one entry is
		// left. Since n must've contained at least two children
		// before deletion (otherwise it would not be a full node) n
		// can never be reduced to nil.
		//
		// When the loop is done, pos contains the index of the single
		// value that is left in n or -2 if n contains at least two
		// values.
		pos := -1
		for i, cld := range &n.Children {
			if cld != nil {
				if pos == -1 {
					pos = i
				} else {
					pos = -2
					break
				}
			}
		}
		if pos >= 0 {
			if pos != 16 {
				// If the remaining entry is a short node, it replaces
				// n and its key gets the missing nibble tacked to the
				// front. This avoids creating an invalid
				// shortNode{..., shortNode{...}}.  Since the entry
				// might not be loaded yet, resolve it just for this
				// check.
				n := n.Children[pos]
				if hash_n, is := n.(hashNode); is {
					if resolved_n, err := self.resolve(hash_n, key_hex_prefix); err != nil {
						return false, nil, err
					} else {
						n = resolved_n
					}
				}
				if cnode, ok := n.(*shortNode); ok {
					k := append([]byte{byte(pos)}, cnode.Key...)
					return true, &shortNode{k, cnode.Val, self.newFlag()}, nil
				}
			}
			// Otherwise, n is replaced by a one-nibble short node
			// containing the child.
			return true, &shortNode{[]byte{byte(pos)}, n.Children[pos], self.newFlag()}, nil
		}
		// n still contains at least two values and cannot be reduced.
		return true, n, nil
	case valueNode:
		return true, nil, nil
	case nil:
		return false, nil, nil
	case hashNode:
		// We've hit a part of the trie that isn't loaded yet. Load
		// the node and delete from it. This leaves all child nodes on
		// the path to the value in the trie.
		rn, err := self.resolve(n, key_hex_prefix)
		if err != nil {
			return false, nil, err
		}
		dirty, nn, err := self.mpt_del(rn, key_hex_prefix, key_hex_rest)
		if !dirty || err != nil {
			return false, rn, err
		}
		return true, nn, nil
	default:
		panic(fmt.Sprintf("%T: invalid node: %v (%v)", n, n, key_hex_rest))
	}
}

func (self *Trie) store(hash hashNode, n node, n_enc []byte) error {
	// TODO
	//enc, err := rlp.EncodeToBytes(self.logicalToStorageRepr(n))
	//if err != nil {
	//	return err
	//}
	switch n.(type) {
	case *fullNode, *shortNode:
	default:
		panic("impossible")
	}
	return self.db.Put(hash, n_enc)
}

func (self *Trie) resolve(hash hashNode, mpt_key_hex_prefix []byte) (node, error) {
	cacheMissCounter.Inc(1)
	enc, err := self.db.Get(hash)
	if enc == nil {
		return nil, &MissingNodeError{NodeHash: hash, Path: mpt_key_hex_prefix}
	}
	if err != nil {
		return nil, err
	}
	ret := mustDecodeNode(common.CopyBytes(mpt_key_hex_prefix), hash, enc, self.cachegen, func(key, value []byte) valueNode {
		_ = hexToKeybytes(key)
		// TODO
		//util.Assert(len(value) == 1)
		//mpt_key := concat(mpt_key_hex_prefix, mpt_key_hex_rest...)
		//ret, err := self.db.Get(mpt_key)
		//util.PanicIfNotNil(err)
		//util.Assert(len(ret) != 0)
		//return ret
		return value
	})
	return ret, nil
}

func (self *Trie) hashRoot(store hasher_store_strategy) (common.Hash, node, error) {
	if self.root == nil {
		return emptyRoot, nil, nil
	}
	hasher := newHasher(self.cachegen, self.cachelimit)
	hasher.dot_g = self.Dot_g
	defer returnHasherToPool(hasher)
	root_hash, root, err := hasher.hash(self.root, true, store)
	return common.BytesToHash(root_hash.(hashNode)), root, err
}

func (self *Trie) newFlag() nodeFlag {
	return nodeFlag{dirty: true, gen: self.cachegen}
}
