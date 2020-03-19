package util

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"golang.org/x/crypto/sha3"
	"hash"
)

type Hasher struct {
	state hash_state
	out   []byte
}
type hash_state interface {
	hash.Hash
	Read([]byte) (int, error)
}

func NewHasher() *Hasher {
	return &Hasher{sha3.NewLegacyKeccak256().(hash_state), make([]byte, common.HashLength)}
}

func (self *Hasher) Write(b ...byte) {
	self.state.Write(b)
}

func (self *Hasher) Hash() []byte {
	self.state.Read(self.out)
	return self.out
}

func (self *Hasher) Reset() {
	self.state.Reset()
	self.out = make([]byte, common.HashLength)
}

var hashers = func() chan *Hasher {
	ret := make(chan *Hasher, 512)
	for i := 0; i < cap(ret); i++ {
		ret <- NewHasher()
	}
	return ret
}()

func Keccak256Pooled(bs ...[]byte) (ret []byte) {
	hasher := <-hashers
	for _, b := range bs {
		hasher.Write(b...)
	}
	ret = hasher.Hash()
	go func() {
		// TODO maybe this an overkill
		hasher.Reset()
		hashers <- hasher
	}()
	return
}
