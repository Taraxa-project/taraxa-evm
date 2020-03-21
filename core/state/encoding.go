package state

import (
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
)

// TODO hide enc/dec behind an interface and use it to delay blocking on storage tries hashing
// refactor account enc/dec

func dec_uint64(b []byte) uint64 {
	switch len(b) {
	case 0:
		return 0
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(b[0])<<8 | uint64(b[1])
	case 3:
		return uint64(b[0])<<16 | uint64(b[1])<<8 | uint64(b[2])
	case 4:
		return uint64(b[0])<<24 | uint64(b[1])<<16 | uint64(b[2])<<8 | uint64(b[3])
	case 5:
		return uint64(b[0])<<32 | uint64(b[1])<<24 | uint64(b[2])<<16 | uint64(b[3])<<8 |
			uint64(b[4])
	case 6:
		return uint64(b[0])<<40 | uint64(b[1])<<32 | uint64(b[2])<<24 | uint64(b[3])<<16 |
			uint64(b[4])<<8 | uint64(b[5])
	case 7:
		return uint64(b[0])<<48 | uint64(b[1])<<40 | uint64(b[2])<<32 | uint64(b[3])<<24 |
			uint64(b[4])<<16 | uint64(b[5])<<8 | uint64(b[6])
	case 8:
		return uint64(b[0])<<56 | uint64(b[1])<<48 | uint64(b[2])<<40 | uint64(b[3])<<32 |
			uint64(b[4])<<24 | uint64(b[5])<<16 | uint64(b[6])<<8 | uint64(b[7])
	}
	panic("impossible")
}

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
	// TODO there's reuse potential between the two encodings
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
