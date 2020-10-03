package state_db

import (
	"math/big"
	"sync"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/util_rlp"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

var MainTrieReader = trie.Reader{MainTrieSchema{}}

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
		encoder.AppendString(state_common.EmptyRLPListHash[:])
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

type Account struct {
	Nonce           uint64
	Balance         *big.Int
	StorageRootHash *common.Hash
	CodeHash        *common.Hash
	CodeSize        uint64
}

func (self *Account) DecodeStorageRepr(enc_storage []byte) {
	rest, tmp := rlp.MustSplitList(enc_storage)
	tmp, rest = rlp.MustSplitSring(rest)
	self.Nonce = bin.DEC_b_endian_compact_64(tmp)
	tmp, rest = rlp.MustSplitSring(rest)
	self.Balance = bigutil.FromBytes(tmp)
	if tmp, rest = rlp.MustSplitSring(rest); len(tmp) != 0 {
		self.StorageRootHash = new(common.Hash).SetBytes(tmp)
	}
	if tmp, rest = rlp.MustSplitSring(rest); len(tmp) != 0 {
		self.CodeHash = new(common.Hash).SetBytes(tmp)
	}
	tmp, rest = rlp.MustSplitSring(rest)
	self.CodeSize = bin.DEC_b_endian_compact_64(tmp)
}

func (self *Account) EncodeForTrie() (enc_storage, enc_hash []byte) {
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
		encoder.AppendString(state_common.EmptyRLPListHash[:])
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

func StorageRoot(acc_enc_storage []byte) *common.Hash {
	return keccak256.HashView(util_rlp.RLPListAt(acc_enc_storage, 2))
}

func CodeHash(acc_enc_storage []byte) *common.Hash {
	return keccak256.HashView(util_rlp.RLPListAt(acc_enc_storage, 3))
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

type MainTrieReadTxn struct {
	Tx ReadTx
}

func (self MainTrieReadTxn) GetValue(key *common.Hash, cb func(v []byte)) {
	self.Tx.Get(COL_main_trie_value, key, cb)
}

func (self MainTrieReadTxn) GetNode(node_hash *common.Hash, cb func([]byte)) {
	self.Tx.Get(COL_main_trie_node, node_hash, cb)
}

type MainTrieWriteTxn struct {
	Tx WriteTx
}

func (self MainTrieWriteTxn) GetValue(key *common.Hash, cb func(v []byte)) {
	self.Tx.Get(COL_main_trie_value, key, cb)
}

func (self MainTrieWriteTxn) GetNode(node_hash *common.Hash, cb func([]byte)) {
	self.Tx.Get(COL_main_trie_node, node_hash, cb)
}

func (self MainTrieWriteTxn) PutValue(key *common.Hash, v []byte) {
	self.Tx.Put(COL_main_trie_value, key, v)
}

func (self MainTrieWriteTxn) PutNode(node_hash *common.Hash, node []byte) {
	self.Tx.Put(COL_main_trie_node, node_hash, node)
}
