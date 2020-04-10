package trie

import "github.com/Taraxa-project/taraxa-evm/common"

type node interface {
	get_hash() *node_hash
}

type full_node struct {
	children [16]node
	hash     *node_hash
}

func (self *full_node) get_hash() *node_hash { return self.hash }

type short_node struct {
	key_part []byte
	val      node
	hash     *node_hash
}

func (self *short_node) get_hash() *node_hash { return self.hash }

type node_hash common.Hash

func (self *node_hash) common_hash() *common.Hash { return (*common.Hash)(self) }
func (self *node_hash) get_hash() *node_hash      { return self }

type value_node struct{ Value }

// TODO ugly
func (self value_node) get_hash() *node_hash { panic("N/A") }

type RawValue struct {
	ENC_storage []byte
	ENC_hash    []byte
}

func (self RawValue) EncodeForTrie() (enc_storage, enc_hash []byte) {
	enc_storage, enc_hash = self.ENC_storage, self.ENC_hash
	return
}
