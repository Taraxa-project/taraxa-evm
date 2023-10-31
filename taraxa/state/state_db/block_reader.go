package state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type ExtendedReader struct{ Reader }

func (er ExtendedReader) GetCode(hash *common.Hash) (ret []byte) {
	er.Get(COL_code, hash, func(bytes []byte) {
		ret = common.CopyBytes(bytes)
	})
	return
}

func (er ExtendedReader) GetAccountStorage(addr *common.Address, key *common.Hash, cb func([]byte)) {
	AccountTrieInputAdapter{addr, er}.GetValue(keccak256.Hash(key[:]), cb)
}

func (er ExtendedReader) GetAccount(addr *common.Address, cb func(Account)) {
	er.GetRawAccount(addr, func(v []byte) {
		cb(DecodeAccountFromTrie(v))
	})
}

func (er ExtendedReader) GetRawAccount(addr *common.Address, cb func([]byte)) {
	MainTrieInputAdapter{er}.GetValue(keccak256.Hash(addr[:]), cb)
}

func (er ExtendedReader) GetCodeByAddress(addr *common.Address) (ret []byte) {
	er.GetRawAccount(addr, func(acc []byte) {
		if code_hash := CodeHash(acc); code_hash != nil {
			ret = er.GetCode(code_hash)
		}
	})
	return
}

func (er ExtendedReader) ForEachStorage(addr *common.Address, f func(*common.Hash, []byte)) {
	er.GetRawAccount(addr, func(acc []byte) {
		storage_root := StorageRoot(acc)
		if storage_root == nil {
			return
		}
		trie.Reader{Schema: AccountTrieSchema{}}.ForEach(
			AccountTrieInputAdapter{addr, er},
			storage_root,
			true,
			func(hash *common.Hash, val trie.Value) {
				enc_storage, _ := val.EncodeForTrie()
				f(hash, enc_storage)
			})
	})
}

func (er ExtendedReader) GetProof(addr *common.Address, key *common.Hash) (res [][]byte, err error) {
	er.GetRawAccount(addr, func(acc []byte) {
		storage_root := StorageRoot(acc)
		if storage_root == nil {
			return
		}
		res, err = trie.Reader{Schema: AccountTrieSchema{}}.Prove(
			AccountTrieInputAdapter{addr, er},
			storage_root,
			keccak256.Hash(key.Bytes()).Bytes())
	})
	return
}

func (er ExtendedReader) GetStorageProof(storage_root *common.Hash, addr *common.Address) (res [][]byte, err error) {
	if storage_root == nil {
		return
	}
	return trie.Reader{Schema: MainTrieSchema{}}.Prove(
		MainTrieInputAdapter{er},
		storage_root,
		keccak256.Hash(addr[:]).Bytes())
}

func (er ExtendedReader) ForEachAccountNodeHashByRoot(storage_root *common.Hash, f func(*common.Hash, []byte)) {
	if storage_root == nil {
		return
	}
	no_addr := common.Address{}
	trie.Reader{Schema: AccountTrieSchema{}}.ForEachNodeHash(
		AccountTrieInputAdapter{&no_addr, er},
		storage_root,
		f)
}

func (er ExtendedReader) ForEachMainNodeHashByRoot(storage_root *common.Hash, f func(*common.Hash, []byte)) {
	if storage_root == nil {
		return
	}
	trie.Reader{Schema: MainTrieSchema{}}.ForEachNodeHash(
		MainTrieInputAdapter{er},
		storage_root,
		f)
}
