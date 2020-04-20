package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

type MainTrieSchema struct{}

func (MainTrieSchema) ValueStorageToHashEncoding(enc_storage []byte) (enc_hash []byte) {
	encoder := take_acc_encoder()
	rlp_list := encoder.ListStart()
	next, curr, err := rlp.SplitList(enc_storage)
	util.PanicIfNotNil(err)
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	encoder.AppendString(curr)
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	encoder.AppendString(curr)
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	if len(curr) != 0 {
		encoder.AppendString(curr)
	} else {
		encoder.AppendString(EmptyRLPListHash[:])
	}
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	if len(curr) != 0 {
		encoder.AppendString(curr)
	} else {
		encoder.AppendString(crypto.EmptyBytesKeccak256[:])
	}
	encoder.ListEnd(rlp_list)
	enc_hash = encoder.ToBytes(-1)
	return_acc_encoder(encoder)
	return
}

func (MainTrieSchema) MaxValueSizeToStoreInTrie() int { return 8 }
