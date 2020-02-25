// Copyright 2015 The go-ethereum Authors
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
//
//import (
//	"bytes"
//	"fmt"
//
//	"github.com/Taraxa-project/taraxa-evm/common"
//	"github.com/Taraxa-project/taraxa-evm/crypto"
//	"github.com/Taraxa-project/taraxa-evm/ethdb"
//	"github.com/Taraxa-project/taraxa-evm/log"
//	"github.com/Taraxa-project/taraxa-evm/rlp"
//)
//
//func (self *Trie) Prove(key []byte, fromLevel uint, proofDb ethdb.Putter) error {
//	mpt_key, err_0 := self.storage_strat.OriginKeyToMPTKey(key)
//	if err_0 != nil {
//		return err_0
//	}
//	// Collect all nodes on the path to key.
//	mpt_key_hex := keybytesToHex(mpt_key)
//	nodes := []node{}
//	tn := self.root
//	for len(mpt_key_hex) > 0 && tn != nil {
//		switch n := tn.(type) {
//		case *shortNode:
//			if len(mpt_key_hex) < len(n.Key) || !bytes.Equal(n.Key, mpt_key_hex[:len(n.Key)]) {
//				// The trie doesn't contain the key.
//				tn = nil
//			} else {
//				tn = n.Val
//				mpt_key_hex = mpt_key_hex[len(n.Key):]
//			}
//			nodes = append(nodes, n)
//		case *fullNode:
//			tn = n.Children[mpt_key_hex[0]]
//			mpt_key_hex = mpt_key_hex[1:]
//			nodes = append(nodes, n)
//		case hashNode:
//			var err error
//			tn, err = self.resolve(n, nil)
//			if err != nil {
//				log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
//				return err
//			}
//		default:
//			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
//		}
//	}
//	hasher := newHasher(0, 0)
//	defer returnHasherToPool(hasher)
//	for i, n := range nodes {
//		// Don't bother checking for errors here since hasher panics
//		// if encoding doesn't work and we're not writing to any database.
//		n, _, _ = hasher.hashChildren(n, nil)
//		hn, _ := hasher.hash_and_maybe_store(n, false, nil)
//		if hash, ok := hn.(hashNode); ok || i == 0 {
//			// If the node's database encoding is a hash (or is the
//			// root node), it becomes a proof element.
//			if fromLevel > 0 {
//				fromLevel--
//			} else {
//				enc, _ := rlp.EncodeToBytes(n)
//				if !ok {
//					hash = crypto.Keccak256(enc)
//				}
//				proofDb.Put(hash, enc)
//			}
//		}
//	}
//	return nil
//}
//
//func VerifyProof(rootHash common.Hash, key []byte, proofDb ethdb.Getter) (value []byte, nodes int) {
//	key = keybytesToHex(key)
//	wantHash := rootHash
//	for i := 0; ; i++ {
//		buf, _ := proofDb.Get(wantHash[:])
//		if buf == nil {
//			panic(fmt.Errorf("proof node %d (hash %064x) missing", i, wantHash))
//		}
//		n := decodeNode(nil, wantHash[:], buf, 0, func([]byte) valueNode {
//			return nil
//		})
//		keyrest, cld := get(n, key)
//		switch cld := cld.(type) {
//		case nil:
//			// The trie doesn't contain the key.
//			return nil, i
//		case hashNode:
//			key = keyrest
//			copy(wantHash[:], cld)
//		case valueNode:
//			return cld, i + 1
//		}
//	}
//}
//
//func get(tn node, key []byte) ([]byte, node) {
//	for {
//		switch n := tn.(type) {
//		case *shortNode:
//			if len(key) < len(n.Key) || !bytes.Equal(n.Key, key[:len(n.Key)]) {
//				return nil, nil
//			}
//			tn = n.Val
//			key = key[len(n.Key):]
//		case *fullNode:
//			tn = n.Children[key[0]]
//			key = key[1:]
//		case hashNode:
//			return key, n
//		case nil:
//			return key, nil
//		case valueNode:
//			return nil, n
//		default:
//			panic(fmt.Sprintf("%T: invalid node: %v", tn, tn))
//		}
//	}
//}
