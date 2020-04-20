package util

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"golang.org/x/crypto/sha3"
	"hash"
	"runtime"
)

type Hasher struct {
	state hash_state
	out   *common.Hash
}
type hash_state interface {
	hash.Hash
	Read([]byte) (int, error)
}

func (self *Hasher) Write(b ...byte) {
	self.state.Write(b)
}

func (self *Hasher) Hash() *common.Hash {
	self.state.Read(self.out[:])
	return self.out
}

func (self *Hasher) Reset() {
	self.state.Reset()
	self.out = new(common.Hash)
}

var hashers_resetter SingleThreadExecutor
var hashers = func() chan *Hasher {
	// TODO configurable size
	ret := make(chan *Hasher, runtime.NumCPU()*128)
	for i := 0; i < cap(ret); i++ {
		ret <- &Hasher{sha3.NewLegacyKeccak256().(hash_state), new(common.Hash)}
	}
	return ret
}()

func GetHasherFromPool() *Hasher {
	return <-hashers
}

func ReturnHasherToPool(hasher *Hasher) {
	hashers_resetter.Do(func() {
		hasher.Reset()
		hashers <- hasher
	})
}

func Hash(bs ...[]byte) (ret *common.Hash) {
	hasher := GetHasherFromPool()
	for _, b := range bs {
		hasher.Write(b...)
	}
	ret = hasher.Hash()
	ReturnHasherToPool(hasher)
	return
}

func HashOnStack(bs ...[]byte) (ret common.Hash) {
	hasher := GetHasherFromPool()
	for _, b := range bs {
		hasher.Write(b...)
	}
	hasher.state.Read(ret[:])
	hashers_resetter.Do(func() {
		hasher.state.Reset()
		hashers <- hasher
	})
	return
}
