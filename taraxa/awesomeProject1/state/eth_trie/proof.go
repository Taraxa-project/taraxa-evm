package eth_trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/trie"
)

type Proof struct {
	*ethdb.MemDatabase
}

func (self *Proof) Verify(digest, key []byte) (value []byte, err error) {
	value, _, err = trie.MustVerifyProof(common.BytesToHash(digest), key, self)
	return
}
