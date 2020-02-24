package trie

// TODO cache rlp

import (
	"bytes"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/metrics"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/binary"
	"github.com/emicklei/dot"
	"io"
)

type Trie struct {
	db                   Database
	root                 node
	cachegen, cachelimit uint16
	Dot_g                *dot.Graph
	storage_strat        StorageStrategy
	async_tasks          chan func()
}
type StorageStrategy = interface {
	OriginKeyToMPTKey(key []byte) (mpt_key []byte, err error)
	MPTKeyToFlat(mpt_key []byte) (flat_key []byte, err error)
}

func New(root_hash *common.Hash, db Database, cachelimit uint16, storage_strat StorageStrategy) (*Trie, error) {
	util.Assert(db != nil)
	if storage_strat == nil {
		storage_strat = DefaultStorageStrategy(0)
	}
	self := &Trie{
		db:            db,
		cachelimit:    cachelimit,
		storage_strat: storage_strat,
		async_tasks:   make(chan func(), 256),
	}
	if root_hash := *root_hash; root_hash != common.ZeroHash && root_hash != EmptyRLPListHash {
		rootnode, err := self.resolve(root_hash[:], nil)
		util.PanicIfNotNil(err)
		self.root = rootnode
	}
	go func() {
		for {
			t, ok := <-self.async_tasks
			if !ok {
				break
			}
			t()
		}
	}()
	return self, nil
}

func (self *Trie) Close() {
	close(self.async_tasks)
}

func (self *Trie) Get(key []byte) ([]byte, error) {
	mpt_key, err_0 := self.storage_strat.OriginKeyToMPTKey(key)
	util.PanicIfNotNil(err_0)
	flat_key, err_1 := self.storage_strat.MPTKeyToFlat(mpt_key)
	util.PanicIfNotNil(err_1)
	flat_v, err_2 := self.db.Get(flat_key)
	util.PanicIfNotNil(err_2)
	return flat_v, nil
	//mpt_key_hex := keybytesToHex(mpt_key)
	//value, newroot, didResolve, err_2 := self.mpt_get(self.root, mpt_key_hex, 0)
	//if err_2 != nil {
	//	return nil, err_2
	//}
	//if didResolve {
	//	self.root = newroot
	//}
	//return value, nil
}

func (self *Trie) InsertAsync(key, value []byte) {
	self.async_tasks <- func() {
		mpt_key, err_0 := self.storage_strat.OriginKeyToMPTKey(key)
		util.PanicIfNotNil(err_0)
		mpt_key_hex := keybytesToHex(mpt_key)
		if len(value) != 0 {
			_, n, err := self.mpt_insert(self.root, nil, mpt_key_hex, valueNode(value))
			util.PanicIfNotNil(err)
			self.root = n
		} else {
			_, n, err := self.mpt_del(self.root, nil, mpt_key_hex)
			util.PanicIfNotNil(err)
			self.root = n
		}
		flat_key, err_1 := self.storage_strat.MPTKeyToFlat(mpt_key)
		util.PanicIfNotNil(err_1)
		util.PanicIfNotNil(self.db.Put(flat_key, value))
	}
}

func (self *Trie) DeleteAsync(key []byte) {
	self.InsertAsync(key, nil)
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
			dirty, nn, err := self.mpt_insert(
				n.Val,
				append(key_hex_prefix, key_hex_rest[:matchlen]...),
				key_hex_rest[matchlen:],
				value,
			)
			if !dirty || err != nil {
				return false, n, err
			}
			return true, &shortNode{n.Key, nn, self.newFlag()}, nil
		}
		// Otherwise branch out at the index where they differ.
		branch := &fullNode{flags: self.newFlag()}
		var err error
		_, branch.Children[n.Key[matchlen]], err = self.mpt_insert(
			nil,
			append(key_hex_prefix, n.Key[:matchlen+1]...),
			n.Key[matchlen+1:],
			n.Val,
		)
		if err != nil {
			return false, nil, err
		}
		_, branch.Children[key_hex_rest[matchlen]], err = self.mpt_insert(
			nil,
			append(key_hex_prefix, key_hex_rest[:matchlen+1]...),
			key_hex_rest[matchlen+1:],
			value,
		)
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
		dirty, nn, err := self.mpt_insert(
			n.Children[key_hex_rest[0]],
			append(key_hex_prefix, key_hex_rest[0]),
			key_hex_rest[1:],
			value,
		)
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
		dirty, child, err := self.mpt_del(
			n.Val,
			append(key_hex_prefix, key_hex_rest[:len(n.Key)]...),
			key_hex_rest[len(n.Key):],
		)
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
			return true, &shortNode{binary.Concat(n.Key, child.Key...), child.Val, self.newFlag()}, nil
		default:
			return true, &shortNode{n.Key, child, self.newFlag()}, nil
		}
	case *fullNode:
		dirty, nn, err := self.mpt_del(
			n.Children[key_hex_rest[0]],
			append(key_hex_prefix, key_hex_rest[0]),
			key_hex_rest[1:],
		)
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
					if resolved_n, err := self.resolve(hash_n, append(key_hex_prefix, byte(pos))); err != nil {
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

func (self *Trie) enc_full(n *fullNode, w io.Writer) error {
	var nodes [16]node
	for i := range nodes {
		if child := n.Children[i]; child != nil {
			nodes[i] = child
		} else {
			nodes[i] = nilValueNode
		}
	}
	return rlp.Encode(w, nodes)
}

func (self *Trie) enc_short(n *shortNode, w io.Writer) error {
	if _, is := n.Val.(valueNode); is {
		return rlp.Encode(w, []interface{}{n.Key})
	}
	util.Assert(n.Val != nil)
	return rlp.Encode(w, []interface{}{n.Key, n.Val})
}

func (self *Trie) store(hash hashNode, n node, _ []byte) error {
	buf, err := rlp.EncodeToBytes(n, self)
	util.PanicIfNotNil(err)
	return self.db.Put(common.CopyBytes(hash), buf)
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
	ret := mustDecodeNode(common.CopyBytes(mpt_key_hex_prefix), hash, enc, self.cachegen, func(mpt_key_hex []byte) valueNode {
		mpt_key := hexToKeybytes(mpt_key_hex)
		flat_key, err_0 := self.storage_strat.MPTKeyToFlat(mpt_key)
		util.PanicIfNotNil(err_0)
		ret, err_1 := self.db.Get(flat_key)
		util.PanicIfNotNil(err_1)
		return ret
	})
	return ret, nil
}

func (self *Trie) hashRoot(store hasher_store_strategy) (ret common.Hash, n node, err error) {
	done := make(chan byte, 1)
	self.async_tasks <- func() {
		defer close(done)
		if self.root == nil {
			ret = EmptyRLPListHash
			return
		}
		hasher := newHasher(self.cachegen, self.cachelimit)
		defer returnHasherToPool(hasher)
		hasher.dot_g = self.Dot_g
		var hash_node node
		hash_node, n, err = hasher.hash(self.root, true, store)
		ret = common.BytesToHash(hash_node.(hashNode))
	}
	<-done
	return
}

func (self *Trie) newFlag() nodeFlag {
	return nodeFlag{dirty: true, gen: self.cachegen}
}

var EmptyRLPListHash = func() common.Hash {
	b, err := rlp.EncodeToBytes([]byte(nil))
	util.PanicIfNotNil(err)
	return crypto.Keccak256Hash(b)
}()

var cacheMissCounter = metrics.NewRegisteredCounter("trie/cachemiss", nil)
var cacheUnloadCounter = metrics.NewRegisteredCounter("trie/cacheunload", nil)

func CacheMisses() int64 {
	return cacheMissCounter.Count()
}

func CacheUnloads() int64 {
	return cacheUnloadCounter.Count()
}
