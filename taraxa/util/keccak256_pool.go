package util

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"golang.org/x/crypto/sha3"
	"hash"
)

type hasher = struct {
	h   hash.Hash
	buf common.Hash
}

var hashers = func() chan *hasher {
	ret := make(chan *hasher, 512)
	for i := 0; i < cap(ret); i++ {
		ret <- &hasher{h: sha3.NewLegacyKeccak256()}
	}
	return ret
}()

func Keccak256Pooled(bs ...[]byte) (ret []byte, ret_release func()) {
	hasher := <-hashers
	for _, b := range bs {
		hasher.h.Write(b)
	}
	return hasher.h.Sum(hasher.buf[:0]), func() {
		hasher.h.Reset()
		hashers <- hasher
	}
}
