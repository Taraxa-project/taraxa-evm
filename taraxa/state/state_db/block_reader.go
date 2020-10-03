package state_db

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type BlockReader struct {
	Tx ReadTx
}

func (self *BlockReader) NotifyDone() {
	self.Tx.NotifyDoneReading()
	self.Tx = nil
}

func (self BlockReader) GetCode(hash *common.Hash) (ret []byte) {
	self.Tx.Get(COL_code, hash, func(bytes []byte) {
		ret = common.CopyBytes(bytes)
	})
	return
}

func (self BlockReader) GetAccountStorage(addr *common.Address, key *common.Hash, cb func([]byte)) {
	AccountTrieReadTxn{addr, self.Tx}.GetValue(keccak256.Hash(key[:]), cb)
}

func (self BlockReader) GetRawAccount(addr *common.Address, cb func([]byte)) {
	MainTrieReadTxn{self.Tx}.GetValue(keccak256.Hash(addr[:]), cb)
}

func (self BlockReader) GetCodeByAddress(addr *common.Address) (ret []byte) {
	self.GetRawAccount(addr, func(acc []byte) {
		if code_hash := CodeHash(acc); code_hash != nil {
			ret = self.GetCode(code_hash)
		}
	})
	return
}

func (self BlockReader) ForEachStorage(addr *common.Address, f func(*common.Hash, []byte)) {
	self.GetRawAccount(addr, func(acc []byte) {
		storage_root := StorageRoot(acc)
		if storage_root == nil {
			return
		}
		AccountTrieReader.ForEach(
			AccountTrieReadTxn{addr, self.Tx},
			storage_root,
			true,
			func(hash *common.Hash, val trie.Value) {
				enc_storage, _ := val.EncodeForTrie()
				f(hash, enc_storage)
			})
	})
}

type Proof struct {
	AccountProof  trie.Proof
	StorageProofs []trie.Proof
}

func (self BlockReader) Prove(state_root *common.Hash, addr *common.Address, keys ...common.Hash) (ret Proof) {
	ret.AccountProof = MainTrieReader.Prove(
		MainTrieReadTxn{self.Tx},
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
	acc_txn := AccountTrieReadTxn{addr, self.Tx}
	for i := 0; i < len(keys); i++ {
		ret.StorageProofs[i] = AccountTrieReader.Prove(acc_txn, storage_root, keccak256.Hash(keys[i][:]))
	}
	return
}
