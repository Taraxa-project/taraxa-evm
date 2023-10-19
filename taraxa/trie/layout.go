package trie

import (
	"encoding/json"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/ethereum/go-ethereum/rlp"
)

const MaxDepth = common.HashLength * 2
const HexKeyLen = MaxDepth + 1
const HexKeyCompactLen = common.HashLength + 1

type hex_key = [HexKeyLen]byte
type hex_key_compact = [HexKeyCompactLen]byte

type node interface {
	get_hash() *node_hash
	encode(w rlp.EncoderBuffer, res *Resolver)
	String() string
}

const full_node_child_cnt = 17

type full_node struct {
	children [full_node_child_cnt]node
	hash     *node_hash
}

func (n *full_node) String() string {
	types := make([]string, full_node_child_cnt)
	for i, c := range n.children {
		if c == nil {
			types[i] = "nil"
			continue
		}
		types[i] = c.String()
	}
	tj, _ := json.Marshal(types)
	return string(tj)
}

func (n *full_node) copy() *full_node { copy := *n; return &copy }
func (n *full_node) get_hash() *node_hash {
	return n.hash
}

type short_node struct {
	key_part []byte
	val      node
	hash     *node_hash
}

func (n *short_node) String() string {
	return "[short:" + common.Bytes2Hex(n.key_part) + ":" + n.get_hash().common_hash().Hex() + "|" + n.val.String() + "]"
}
func (n *short_node) copy() *short_node    { copy := *n; return &copy }
func (n *short_node) get_hash() *node_hash { return n.hash }

type node_hash common.Hash

func (n *node_hash) String() string            { return "[hash:" + n.common_hash().Hex() + "]" }
func (n *node_hash) get_hash() *node_hash      { return n }
func (n *node_hash) common_hash() *common.Hash { return (*common.Hash)(n) }

type value_node struct{ val Value }

func (n value_node) String() string {
	if n.val == nil {
		return "[value:nil]"
	}
	v1, v2 := n.val.EncodeForTrie()
	return "[value:" + common.Bytes2Hex(v1) + ":" + common.Bytes2Hex(v2) + "]"
}
func (n value_node) get_hash() *node_hash {
	panic("N/A")
}

var nil_val_node = value_node{nil}

type Value interface {
	EncodeForTrie() (enc_storage, enc_hash []byte)
}

type internal_value struct{ enc_storage, enc_hash []byte }

func (v internal_value) EncodeForTrie() (enc_storage, enc_hash []byte) {
	enc_storage, enc_hash = v.enc_storage, v.enc_hash
	return
}
