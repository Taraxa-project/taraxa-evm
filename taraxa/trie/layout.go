package trie

import "github.com/Taraxa-project/taraxa-evm/common"

const MaxDepth = common.HashLength * 2
const HexKeyLen = MaxDepth + 1
const HexKeyCompactLen = common.HashLength + 1

type hex_key = [HexKeyLen]byte
type hex_key_compact = [HexKeyCompactLen]byte

type node interface {
	get_hash() *node_hash
}

const full_node_child_cnt = 16

type full_node struct {
	children [full_node_child_cnt]node
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

func (self *node_hash) get_hash() *node_hash      { return self }
func (self *node_hash) common_hash() *common.Hash { return (*common.Hash)(self) }

type value_node struct{ val Value }

func (self value_node) get_hash() *node_hash { panic("N/A") }

var nil_val_node = value_node{nil}

type Value interface {
	EncodeForTrie() (enc_storage, enc_hash []byte)
}

type internal_value struct{ enc_storage, enc_hash []byte }

func (self internal_value) EncodeForTrie() (enc_storage, enc_hash []byte) {
	enc_storage, enc_hash = self.enc_storage, self.enc_hash
	return
}
