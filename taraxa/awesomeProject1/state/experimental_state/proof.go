package experimental_state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/Taraxa-project/taraxa-evm/crypto"
)

type Proof [][]byte

var InvalidProof = errors.New("Invalid Proof")

func (self Proof) Verify(expected_digest, key []byte) (value []byte, err error) {
	entry_val := self[0]
	if !bytes.HasPrefix(entry_val, key) {
		return nil, InvalidProof
	}
	entry_val = entry_val[len(key):]
	entry_ordinal := binary.BigEndian.Uint64(entry_val[:EntryOrdinalSize])
	digest := crypto.Keccak256(self[0])
	sibling_hashes_buf := make([][]byte, Arity)
	for _, sibling_hash := range self[1:] {
		entry_local_pos := entry_ordinal % Arity
		sibling_local_pos := 1 - entry_local_pos
		sibling_hashes_buf[entry_local_pos] = digest
		sibling_hashes_buf[sibling_local_pos] = sibling_hash
		digest = crypto.Keccak256(sibling_hashes_buf...)
		entry_ordinal /= Arity
	}
	if bytes.Compare(expected_digest, digest) != 0 {
		return nil, InvalidProof
	}
	return entry_val[EntryOrdinalSize:], nil
}
