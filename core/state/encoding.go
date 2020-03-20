package state

import (
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
)

func dec(o *state_object, enc_storage []byte) *state_object {
	next, curr, err := rlp.SplitList(enc_storage)
	util.PanicIfNotNil(err)
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	o.nonce = dec_uint64(curr)
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	o.balance = new(big.Int).SetBytes(curr)
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	o.storage_root_hash = curr
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	o.code.hash = curr
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	o.code.size = dec_uint64(curr)
	return o
}

func enc(encoder *rlp.Encoder, o *state_object) (enc_storage, enc_hash []byte) {
	storage_rlp_list := encoder.ListStart()
	encoder.AppendUint(o.nonce)
	encoder.AppendBigInt(o.balance)
	encoder.AppendString(o.storage_root_hash)
	encoder.AppendString(o.code.hash)
	encoder.AppendUint(o.code.size)
	encoder.ListEnd(storage_rlp_list)
	enc_storage = encoder.ToBytes(nil)
	encoder.Reset()
	hash_rlp_list := encoder.ListStart()
	encoder.AppendUint(o.nonce)
	encoder.AppendBigInt(o.balance)
	if len(o.storage_root_hash) != 0 {
		encoder.AppendString(o.storage_root_hash)
	} else {
		encoder.AppendString(empty_rlp_list_hash[:])
	}
	if o.code.size != 0 {
		encoder.AppendString(o.code.hash)
	} else {
		encoder.AppendString(crypto.EmptyBytesKeccak256[:])
	}
	encoder.ListEnd(hash_rlp_list)
	enc_hash = encoder.ToBytes(nil)
	encoder.Reset()
	return
}

func enc_storage_2_hash(encoder *rlp.Encoder, enc_storage []byte) (enc_hash []byte) {
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
		encoder.AppendString(empty_rlp_list_hash[:])
	}
	curr, next, err = rlp.SplitString(next)
	util.PanicIfNotNil(err)
	if len(curr) != 0 {
		encoder.AppendString(curr)
	} else {
		encoder.AppendString(crypto.EmptyBytesKeccak256[:])
	}
	encoder.ListEnd(rlp_list)
	enc_hash = encoder.ToBytes(nil)
	encoder.Reset()
	return
}
