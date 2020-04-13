package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
	"sync"
)

var acc_encoder_pool = sync.Pool{New: func() interface{} {
	ret := new(rlp.Encoder)
	ret.ResizeReset(1<<8, 1)
	return ret
}}

func take_acc_encoder() (ret *rlp.Encoder) {
	ret = acc_encoder_pool.Get().(*rlp.Encoder)
	ret.Reset()
	return ret
}

func return_acc_encoder(encoder *rlp.Encoder) {
	acc_encoder_pool.Put(encoder)
}

func acc_trie_value(val *big.Int) SimpleTrieValue {
	return rlp.ToRLPStringSimple(val.Bytes())
}

var empty_rlp_list_hash = func() common.Hash {
	b, err := rlp.EncodeToBytes([]byte(nil))
	util.PanicIfNotNil(err)
	return crypto.Keccak256Hash(b)
}()

type SimpleTrieValue []byte

func (self SimpleTrieValue) EncodeForTrie() (enc_storage, enc_hash []byte) {
	enc_storage, enc_hash = self, self
	return
}
