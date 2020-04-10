package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type Proof = struct {
	Value []byte
	Nodes [][]byte
}

func Prove(schema Schema, root_hash *common.Hash, in Input, key *common.Hash) (ret Proof) {
	return
}

func VerifyProof(schema Schema, root_hash *common.Hash, in Input, key *common.Hash, proof *Proof) bool {
	return true
}
