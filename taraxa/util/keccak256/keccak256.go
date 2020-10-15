package keccak256

import (
	"hash"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"golang.org/x/crypto/sha3"
)

type Hasher struct {
	state              hash_state
	result_buf         *common.Hash
	result_buf_escaped bool
	from_pool          bool
}
type hash_state interface {
	hash.Hash
	Read([]byte) (int, error)
}

func (self *Hasher) Init() *Hasher {
	self.state = sha3.NewLegacyKeccak256().(hash_state)
	self.result_buf = new(common.Hash)
	return self
}

func (self *Hasher) Write(b ...byte) {
	self.state.Write(b)
}

func (self *Hasher) Hash() *common.Hash {
	self.state.Read(self.result_buf[:])
	self.result_buf_escaped = true
	return self.result_buf
}

func (self *Hasher) HashAndReturnByValue() (ret common.Hash) {
	self.state.Read(ret[:])
	return
}

func (self *Hasher) Reset() {
	self.state.Reset()
	if self.result_buf_escaped {
		self.result_buf = new(common.Hash)
		self.result_buf_escaped = false
	}
}

var hashers chan *Hasher
var hashers_expended chan *Hasher
var pool_init_mu sync.Mutex

func init_pool(num_hashers uint64) {
	hashers = make(chan *Hasher, num_hashers)
	hashers_expended = make(chan *Hasher, (num_hashers/2)+1)
	for i := uint64(0); i < num_hashers; i++ {
		hasher := new(Hasher).Init()
		hasher.from_pool = true
		hashers <- hasher
	}
	go func() {
		for {
			hasher := <-hashers_expended
			hasher.Reset()
			hashers <- hasher
		}
	}()
}

func InitPool(size uint64) {
	defer util.LockUnlock(&pool_init_mu)()
	if hashers != nil {
		panic("already initialized: either by you or lazily")
	}
	init_pool(size)
}

func GetHasherFromPool() (ret *Hasher) {
	if hashers == nil {
		pool_init_mu.Lock()
		if hashers == nil {
			init_pool(uint64(runtime.NumCPU()) * 1024) // 0.5 * num_cpu MB
		}
		pool_init_mu.Unlock()
	}
	select {
	case ret = <-hashers:
	default:
		ret = new(Hasher).Init()
	}
	return
}

func ReturnHasherToPool(hasher *Hasher) {
	if hasher.from_pool {
		hashers_expended <- hasher
	}
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

func HashAndReturnByValue(bs ...[]byte) (ret common.Hash) {
	hasher := GetHasherFromPool()
	for _, b := range bs {
		hasher.Write(b...)
	}
	ret = hasher.HashAndReturnByValue()
	ReturnHasherToPool(hasher)
	return
}

func HashView(bytes []byte) (ret *common.Hash) {
	if l := len(bytes); l != 0 && asserts.Holds(l == common.HashLength) {
		ret = (*common.Hash)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&bytes)).Data))
	}
	return
}
