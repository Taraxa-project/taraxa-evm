package state_historical

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type BlockReader struct {
	main_tr_db state_trie.MainTrieDBReadOnly
	db_tx      state_common.BlockReadTransaction
}

func (self *BlockReader) SetTransaction(db_tx state_common.BlockReadTransaction) *BlockReader {
	self.main_tr_db.SetTransaction(db_tx)
	self.db_tx = db_tx
	return self
}

func (self *BlockReader) GetCode(hash *common.Hash) state_common.ManagedSlice {
	return self.db_tx.GetCode(hash)
}

func (self *BlockReader) GetAccountStorage(addr *common.Address, key *common.Hash, cb func([]byte)) {
	self.MakeAccountTrieDB(addr).GetValue(keccak256.Hash(key[:]), cb)
}

func (self *BlockReader) GetRawAccount(addr *common.Address, cb func([]byte)) {
	self.main_tr_db.GetValue(keccak256.Hash(addr[:]), cb)
}

func (self *BlockReader) GetCodeByAddress(addr *common.Address) (ret state_common.ManagedSlice) {
	self.GetRawAccount(addr, func(acc []byte) {
		if code_hash := state_trie.CodeHash(acc); code_hash != nil {
			ret = self.GetCode(code_hash)
		}
	})
	return
}

func (self *BlockReader) MakeAccountTrieDB(addr *common.Address) *state_trie.AccountTrieReadDB {
	return new(state_trie.AccountTrieReadDB).Init(addr).SetTransaction(self.db_tx)
}

func (self *BlockReader) ForEachStorage(addr *common.Address, f func(*common.Hash, []byte)) {
	self.GetRawAccount(addr, func(acc []byte) {
		storage_root := state_trie.StorageRoot(acc)
		if storage_root == nil {
			return
		}
		trie.Reader{self.MakeAccountTrieDB(addr)}.ForEach(storage_root, true, func(hash *common.Hash, val trie.Value) {
			enc_storage, _ := val.EncodeForTrie()
			f(hash, enc_storage)
		})
	})
}

type Proof struct {
	AccountProof  trie.Proof
	StorageProofs []trie.Proof
}

func (self *BlockReader) Prove(state_root *common.Hash, addr *common.Address, keys ...common.Hash) (ret Proof) {
	ret.AccountProof = trie.Reader{&self.main_tr_db}.Prove(state_root, keccak256.Hash(addr[:]))
	if len(ret.AccountProof.Value) == 0 || len(keys) == 0 {
		return
	}
	ret.StorageProofs = make([]trie.Proof, len(keys))
	storage_root := state_trie.StorageRoot(ret.AccountProof.Value)
	if storage_root == nil {
		return
	}
	acc_tr_reader := trie.Reader{self.MakeAccountTrieDB(addr)}
	for i := 0; i < len(keys); i++ {
		ret.StorageProofs[i] = acc_tr_reader.Prove(storage_root, keccak256.Hash(keys[i][:]))
	}
	return
}
