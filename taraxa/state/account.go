package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"math/big"
)

type Account struct {
	nonce             uint64
	balance           *big.Int
	storage_root_hash *common.Hash
	code_hash         *common.Hash
	code_size         uint64
}

func (self *Account) I_from_storage_encoding(enc_storage []byte) (ret *Account) {
	rest, curr, err := rlp.SplitList(enc_storage)
	util.PanicIfNotNil(err)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	self.nonce = bin.DEC_b_endian_compact_64(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	self.balance = new(big.Int).SetBytes(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	self.storage_root_hash = bin.HashView(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	self.code_hash = bin.HashView(curr)
	curr, rest, err = rlp.SplitString(rest)
	util.PanicIfNotNil(err)
	self.code_size = bin.DEC_b_endian_compact_64(curr)
	return self
}

func (self *Account) is_empty() bool {
	return self.nonce == 0 && self.balance.Sign() == 0 && self.code_size == 0
}

func (self *Account) EncodeForTrie() (enc_storage, enc_hash []byte) {
	encoder := take_acc_encoder()
	storage_rlp_list := encoder.ListStart()
	encoder.AppendUint(self.nonce)
	encoder.AppendBigInt(self.balance)
	if self.storage_root_hash != nil {
		encoder.AppendString(self.storage_root_hash[:])
	} else {
		encoder.AppendEmptyString()
	}
	if self.code_hash != nil {
		encoder.AppendString(self.code_hash[:])
	} else {
		encoder.AppendEmptyString()
	}
	encoder.AppendUint(self.code_size)
	encoder.ListEnd(storage_rlp_list)
	enc_storage = encoder.ToBytes(-1)
	encoder.Reset()
	hash_rlp_list := encoder.ListStart()
	encoder.AppendUint(self.nonce)
	encoder.AppendBigInt(self.balance)
	if self.storage_root_hash != nil {
		encoder.AppendString(self.storage_root_hash[:])
	} else {
		encoder.AppendString(empty_rlp_list_hash[:])
	}
	if self.code_hash != nil {
		encoder.AppendString(self.code_hash[:])
	} else {
		encoder.AppendString(crypto.EmptyBytesKeccak256[:])
	}
	encoder.ListEnd(hash_rlp_list)
	enc_hash = encoder.ToBytes(-1)
	return_acc_encoder(encoder)
	return
}
