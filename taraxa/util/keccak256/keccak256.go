package keccak256

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"golang.org/x/crypto/sha3"
	"hash"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
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

var hashers_resetter util.SingleThreadExecutor
var hashers chan *Hasher
var hashers_init_mu sync.Mutex

func InitPool(size uint64) {
	defer util.LockUnlock(&hashers_init_mu)()
	if hashers != nil {
		panic("already initialized")
	}
	init_pool(size)
}

func init_pool(size uint64) {
	hashers = make(chan *Hasher, size)
	for i := uint64(0); i < size; i++ {
		hashers <- &Hasher{sha3.NewLegacyKeccak256().(hash_state), new(common.Hash)}
	}
}

func GetHasherFromPool() *Hasher {
	if hashers == nil {
		defer util.LockUnlock(&hashers_init_mu)()
		if hashers == nil {
			init_pool(uint64(runtime.NumCPU()) * 128)
		}
	}
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

func HashView(bytes []byte) (ret *common.Hash) {
	if l := len(bytes); l != 0 && assert.Holds(l == common.HashLength) {
		ret = (*common.Hash)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&bytes)).Data))
	}
	return
}
