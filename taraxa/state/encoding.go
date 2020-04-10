package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"sync"
)

var acc_encoder_pool = sync.Pool{New: func() interface{} {
	return rlp.NewEncoder(rlp.EncoderConfig{1 << 8, 1})
}}

func take_acc_encoder() (ret *rlp.Encoder) {
	ret = acc_encoder_pool.Get().(*rlp.Encoder)
	ret.Reset()
	return
}

func return_acc_encoder(encoder *rlp.Encoder) {
	acc_encoder_pool.Put(encoder)
}

var empty_rlp_list_hash = func() common.Hash {
	b, err := rlp.EncodeToBytes([]byte(nil))
	util.PanicIfNotNil(err)
	return crypto.Keccak256Hash(b)
}()
