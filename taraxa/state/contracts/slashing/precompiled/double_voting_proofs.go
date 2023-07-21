package slashing

import (
	"bytes"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

type DoubleVotingProof struct {
	ProofAuthor *common.Address
	Block       types.BlockNum
	Vote1Hash   *common.Hash
	Vote2Hash   *common.Hash
	TxHash      *common.Hash // Tx hash with full proof
}

type DoubleVotingProofs struct {
	storage      *contract_storage.StorageWrapper
	proofs_field []byte
}

func (self *DoubleVotingProofs) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.storage = stor
	self.proofs_field = prefix
}

func (self *DoubleVotingProofs) GenDoubleVotingProofDbKey(validator *common.Address, vote1Hash *common.Hash, vote2Hash *common.Hash) (db_key *common.Hash) {
	var smaller_vote_hash *common.Hash
	var greater_vote_hash *common.Hash

	// To create the key, hashes must be sorted to have the same key for both combinations of votes
	cmp_res := bytes.Compare(vote1Hash.Bytes(), vote2Hash.Bytes())
	if cmp_res == -1 {
		smaller_vote_hash = vote1Hash
		greater_vote_hash = vote2Hash
	} else if cmp_res == 1 {
		smaller_vote_hash = vote2Hash
		greater_vote_hash = vote1Hash
	} else {
		panic("Votes hashes are the same")
	}

	// Create the key
	db_key = contract_storage.Stor_k_1(self.proofs_field, smaller_vote_hash.Bytes(), greater_vote_hash.Bytes(), validator.Bytes())
	return
}

// func (self *DoubleVotingProofs) SaveProof(block types.BlockNum, proofAuthor *common.Address, validator *common.Address, vote1Hash *common.Hash, vote2Hash *common.Hash, txHash *common.Hash) (db_key *common.Hash) {
// 	db_key = self.GenDoubleVotingProofDbKey(validator, vote1Hash, vote2Hash)
// 	proof := DoubleVotingProof{proofAuthor, block, vote1Hash, vote2Hash, txHash}

// 	self.storage.Put(db_key, rlp.MustEncodeToBytes(proof))
// 	return
// }

func (self *DoubleVotingProofs) SaveProof(db_key *common.Hash, proof *DoubleVotingProof) {
	self.storage.Put(db_key, rlp.MustEncodeToBytes(proof))
	return
}

func (self *DoubleVotingProofs) GetProof(db_key *common.Hash) (proof *DoubleVotingProof) {
	// TODO: any way how to check for existence withou copying the bytes ???
	self.storage.Get(db_key, func(bytes []byte) {
		proof = new(DoubleVotingProof)
		rlp.MustDecodeBytes(bytes, proof)
	})

	return
}

func (self *DoubleVotingProofs) ProofExists(db_key *common.Hash) bool {
	// TODO: any way how to check for existence without copying the bytes ???
	proof := self.GetProof(db_key)
	if proof != nil {
		return true
	}

	return false
}
