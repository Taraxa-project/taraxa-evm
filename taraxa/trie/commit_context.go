package trie

import (
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

type commit_context struct {
	hex_key_compact_tmp hex_key_compact
	enc_hash            hash_encoder
	enc_storage         rlp.Encoder
}

func (self *commit_context) Reset() {
	self.enc_hash.Reset()
	self.enc_storage.Reset()
}
