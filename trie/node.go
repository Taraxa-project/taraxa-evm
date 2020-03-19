package trie

type node interface {
	get_hash() node_hash
}

type full_node struct {
	children [16]node
	hash     node_hash
}

func (self *full_node) get_hash() node_hash { return self.hash }

type short_node struct {
	key_part []byte
	val      node
	hash     node_hash
}

func (self *short_node) get_hash() node_hash { return self.hash }

type node_hash []byte

func (self node_hash) get_hash() node_hash { return self }

type value struct {
	enc_storage []byte
	enc_hash    []byte
}

// TODO ugly
func (self *value) get_hash() node_hash { panic("N/A") }
