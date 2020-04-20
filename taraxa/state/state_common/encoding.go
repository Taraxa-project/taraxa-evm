package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"math/big"
	"sync"
)

func EncodeAccountTrieValue(val *big.Int) trie.Value {
	return simple_trie_value(rlp.ToRLPStringSimple(val.Bytes()))
}

var EmptyRLPListHash = func() common.Hash {
	return crypto.Keccak256Hash(rlp.MustEncodeToBytes([]byte(nil)))
}()

func DecodeAccount(acc *Account, enc_storage []byte) {
	rest, curr, err := rlp.SplitList(enc_storage)
	util.PanicIfNotNil(err)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	acc.Nonce = bin.DEC_b_endian_compact_64(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	acc.Balance = new(big.Int).SetBytes(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	acc.StorageRootHash = bin.HashView(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	acc.CodeHash = bin.HashView(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	acc.CodeSize = bin.DEC_b_endian_compact_64(curr)
}

type AccountEncoder struct{ *Account }

func (self AccountEncoder) EncodeForTrie() (enc_storage, enc_hash []byte) {
	encoder := take_acc_encoder()
	storage_rlp_list := encoder.ListStart()
	encoder.AppendUint(self.Nonce)
	encoder.AppendBigInt(self.Balance)
	if self.StorageRootHash != nil {
		encoder.AppendString(self.StorageRootHash[:])
	} else {
		encoder.AppendEmptyString()
	}
	if self.CodeHash != nil {
		encoder.AppendString(self.CodeHash[:])
	} else {
		encoder.AppendEmptyString()
	}
	encoder.AppendUint(self.CodeSize)
	encoder.ListEnd(storage_rlp_list)
	enc_storage = encoder.ToBytes(-1)
	encoder.Reset()
	hash_rlp_list := encoder.ListStart()
	encoder.AppendUint(self.Nonce)
	encoder.AppendBigInt(self.Balance)
	if self.StorageRootHash != nil {
		encoder.AppendString(self.StorageRootHash[:])
	} else {
		encoder.AppendString(EmptyRLPListHash[:])
	}
	if self.CodeHash != nil {
		encoder.AppendString(self.CodeHash[:])
	} else {
		encoder.AppendString(crypto.EmptyBytesKeccak256[:])
	}
	encoder.ListEnd(hash_rlp_list)
	enc_hash = encoder.ToBytes(-1)
	return_acc_encoder(encoder)
	return
}

func take_acc_encoder() (ret *rlp.Encoder) {
	ret = acc_encoder_pool.Get().(*rlp.Encoder)
	ret.Reset()
	return ret
}

func return_acc_encoder(encoder *rlp.Encoder) {
	acc_encoder_pool.Put(encoder)
}

var acc_encoder_pool = sync.Pool{New: func() interface{} {
	ret := new(rlp.Encoder)
	ret.ResizeReset(1<<8, 1)
	return ret
}}

type simple_trie_value []byte

func (self simple_trie_value) EncodeForTrie() (enc_storage, enc_hash []byte) {
	enc_storage, enc_hash = self, self
	return
}
