package state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type ExtendedReader struct{ Reader }

func (self ExtendedReader) GetCode(hash *common.Hash) (ret []byte) {
	self.Get(COL_code, hash, func(bytes []byte) {
		ret = common.CopyBytes(bytes)
	})
	return
}

func (self ExtendedReader) GetAccountStorage(addr *common.Address, key *common.Hash, cb func([]byte)) {
	AccountTrieInputAdapter{addr, self}.GetValue(keccak256.Hash(key[:]), cb)
}

func (self ExtendedReader) GetAccount(addr *common.Address, cb func(Account)) {
	self.GetRawAccount(addr, func(v []byte) {
		cb(DecodeAccountFromTrie(v))
	})
}

func (self ExtendedReader) GetRawAccount(addr *common.Address, cb func([]byte)) {
	MainTrieInputAdapter{self}.GetValue(keccak256.Hash(addr[:]), cb)
}

func (self ExtendedReader) GetCodeByAddress(addr *common.Address) (ret []byte) {
	self.GetRawAccount(addr, func(acc []byte) {
		if code_hash := CodeHash(acc); code_hash != nil {
			ret = self.GetCode(code_hash)
		}
	})
	return
}

func (self ExtendedReader) ForEachStorage(addr *common.Address, f func(*common.Hash, []byte)) {
	self.GetRawAccount(addr, func(acc []byte) {
		storage_root := StorageRoot(acc)
		if storage_root == nil {
			return
		}
		trie.Reader{AccountTrieSchema{}}.ForEach(
			AccountTrieInputAdapter{addr, self},
			storage_root,
			true,
			func(hash *common.Hash, val trie.Value) {
				enc_storage, _ := val.EncodeForTrie()
				f(hash, enc_storage)
			})
	})
}

func (self ExtendedReader) ForEachAccountNodeHashByRoot(storage_root *common.Hash, f func(*common.Hash)) {
	if storage_root == nil {
		return
	}
	no_addr := common.Address{}
	trie.Reader{AccountTrieSchema{}}.ForEachNodeHash(
		AccountTrieInputAdapter{&no_addr, self},
		storage_root,
		func(hash *common.Hash) {
			f(hash)
		})
}

func (self ExtendedReader) ForEachMainNodeHashByRoot(storage_root *common.Hash, f func(*common.Hash)) {
	if storage_root == nil {
		return
	}
	trie.Reader{MainTrieSchema{}}.ForEachNodeHash(
		MainTrieInputAdapter{self},
		storage_root,
		func(hash *common.Hash) {
			f(hash)
		})
}

type Proof struct {
	AccountProof  trie.Proof
	StorageProofs []trie.Proof
}

func (self ExtendedReader) Prove(state_root *common.Hash, addr *common.Address, keys ...common.Hash) (ret Proof) {
	ret.AccountProof = trie.Reader{MainTrieSchema{}}.Prove(
		MainTrieInputAdapter{self},
		state_root,
		keccak256.Hash(addr[:]))
	if len(ret.AccountProof.Value) == 0 || len(keys) == 0 {
		return
	}
	ret.StorageProofs = make([]trie.Proof, len(keys))
	storage_root := StorageRoot(ret.AccountProof.Value)
	if storage_root == nil {
		return
	}
	acc_txn := AccountTrieInputAdapter{addr, self}
	for i := 0; i < len(keys); i++ {
		ret.StorageProofs[i] = trie.Reader{AccountTrieSchema{}}.Prove(acc_txn, storage_root, keccak256.Hash(keys[i][:]))
	}
	return
}
