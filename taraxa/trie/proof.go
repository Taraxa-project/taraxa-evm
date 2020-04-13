package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Proof = struct {
	Value []byte
	Nodes [][]byte
}

func Prove(db ReadOnlyDB, root_hash *common.Hash, key *common.Hash) (ret Proof) {
	return
}

func VerifyProof(db ReadOnlyDB, root_hash *common.Hash, key *common.Hash, proof *Proof) bool {
	return true
}
