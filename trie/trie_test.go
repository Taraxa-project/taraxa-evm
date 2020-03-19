package trie

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/rand"
	"testing"
)

type noop_schema struct{}

func (noop_schema) FlatKey(hashed_key []byte) (flat_key []byte) {
	return hashed_key
}

func (noop_schema) StorageToHashEncoding(enc_storage []byte) (enc_hash []byte) {
	return enc_storage
}

func (noop_schema) MaxStorageEncSizeToStoreInTrie() int {
	return 0
}

type noop_storage struct{}

func (noop_storage) PutAsync(col StorageColumn, key, value []byte) {

}

func (noop_storage) DeleteAsync(col StorageColumn, key []byte) {
}

func (noop_storage) GetCommitted(col StorageColumn, key []byte) []byte {
	panic("implement me")
}

func TestOrder(t *testing.T) {
	rand.Seed(0)
	val := []byte{0}
	var keys [20][]byte
	for i := range keys {
		keys[i] = util.RandomBytes(64)
	}
	for i := 0; i < 10; i++ {
		set := make(map[int]bool)
		tr := NewTrie(nil, noop_schema{}, noop_storage{}, 0)
		for _, j := range rand.Perm(len(keys)) {
			set[j] = true
			tr.Put(keys[j], val, val)
		}
		util.Assert(len(set) == len(keys))
		tr.CommitNodes()
		fmt.Println("===========")
	}
}
