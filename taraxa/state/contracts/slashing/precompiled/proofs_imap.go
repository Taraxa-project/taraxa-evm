package slashing

import (
	contract_storage "github.com/Taraxa-project/taraxa-evm/taraxa/state/contracts/storage"
)

// Supported proofs types
type ProofType uint8

const (
	// since iota starts with 0, the first value
	// defined here will be the default
	Undefined ProofType = iota
	DoubleVoting
	// Add new proof types here
)

type ProofKey struct {
	Key        []byte
	Proof_type ProofType
}

// ProofsIMapIMap is a IterableMap wrapper for storing malicious behaviour proofs
type ProofsIMap struct {
	proofs contract_storage.IterableMap
}

// Inits iterbale map with prefix, so multiple iterbale maps can coexists thanks to different prefixes
func (self *ProofsIMap) Init(stor *contract_storage.StorageWrapper, prefix []byte) {
	self.proofs.Init(stor, prefix)
}

// Checks is proof exists in iterable map
func (self *ProofsIMap) ProofExists(proof_type ProofType, key []byte) bool {
	return self.proofs.ItemExists(createProofKey(proof_type, key))
}

// Creates proof from iterable map
func (self *ProofsIMap) CreateProof(proof_type ProofType, key []byte) bool {
	return self.proofs.CreateItem(createProofKey(proof_type, key))
}

// Removes proof from iterable map, returns number of left proofs in the iterbale map
func (self *ProofsIMap) RemoveProof(proof_type ProofType, key []byte) uint32 {
	return self.proofs.RemoveItem(createProofKey(proof_type, key))
}

func (self *ProofsIMap) GetProofs(batch uint32, count uint32) (result []ProofKey, end bool) {
	items, end := self.proofs.GetItems(batch, count)

	result = make([]ProofKey, len(items))
	for idx := 0; idx < len(items); idx++ {
		proof_key_bytes := items[idx]
		// Parse last byte of proof_key to get proof type
		proof_type := ProofType(proof_key_bytes[len(proof_key_bytes)-1])
		// Remove last byte to get actual key
		key := proof_key_bytes[:len(proof_key_bytes)-1]

		result[idx] = ProofKey{key, proof_type}
	}

	return
}

// Returns number of stored items
func (self *ProofsIMap) GetCount() (count uint32) {
	return self.proofs.GetCount()
}

// Proof key consists of key + proof type. Proof type is always last byte of proof key
func createProofKey(proof_type ProofType, key []byte) []byte {
	return append(key, byte(proof_type))
}
