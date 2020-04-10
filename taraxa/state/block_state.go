package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"math/big"
)

type BlockState struct {
	db      DB
	blk_num types.BlockNum
}

func (self *BlockState) GetCode(code_hash *common.Hash) (ret []byte) {
	ret = self.db.GetCode(code_hash)
	return
}

func (self *BlockState) GetAccount(addr *common.Address) (ret Account, present bool) {
	enc_storage := self.GetAccountStorageEncoding(addr)
	if present = len(enc_storage) != 0; present {
		ret.I_FromStorageEncoding(enc_storage)
	}
	return
}

func (self *BlockState) GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int {
	if enc_storage := self.GetStorage(addr, key); len(enc_storage) != 0 {
		return new(big.Int).SetBytes(enc_storage)
	}
	return common.Big0
}

type Proof = struct {
	AccountProof  trie.Proof
	StorageProofs []trie.Proof
}

func (self *BlockState) Prove(state_root *common.Hash, addr *common.Address, keys ...common.Hash) (ret Proof) {
	addr_hash := util.HashOnStack(addr[:])
	ret.AccountProof = trie.Prove(MainTrieSchema{}, state_root, &MainTrieInput{*self}, &addr_hash)
	if len(ret.AccountProof.Value) == 0 || len(keys) == 0 {
		return
	}
	ret.StorageProofs = make([]trie.Proof, len(keys))
	storage_root, err := util.RLPListAt(ret.AccountProof.Value, 2)
	util.PanicIfNotNil(err)
	if len(storage_root) == 0 {
		return
	}
	acc_tr_input_reader := &AccountTrieInput{*self, addr}
	storage_root_h := bin.HashView(storage_root)
	for i := 0; i < len(keys); i++ {
		key_hash := util.HashOnStack(keys[i][:])
		ret.StorageProofs[i] = trie.Prove(AccountTrieSchema{}, storage_root_h, acc_tr_input_reader, &key_hash)
	}
	return
}

func (self *BlockState) GetAccountStorageEncoding(addr *common.Address) []byte {
	return self.db.GetMainTrieValue(self.blk_num, util.Hash(addr[:]))
}

func (self *BlockState) GetAccountEthEncoding(addr *common.Address) (ret []byte) {
	if val := self.GetAccountStorageEncoding(addr); len(val) != 0 {
		ret = MainTrieSchema{}.ValueStorageToHashEncoding(val)
	}
	return
}

func (self *BlockState) GetStorage(addr *common.Address, key *common.Hash) (ret []byte) {
	key_hash := util.HashOnStack(key[:])
	if ret = self.db.GetAccountTrieValue(self.blk_num, addr, &key_hash); len(ret) == 0 {
		return
	}
	_, ret, _, err := rlp.Split(ret)
	util.PanicIfNotNil(err)
	return
}

func (self *BlockState) GetCodeByAddress(addr *common.Address) (ret []byte) {
	acc := self.GetAccountStorageEncoding(addr)
	if len(acc) == 0 {
		return
	}
	code_hash, err := util.RLPListAt(acc, 3)
	util.PanicIfNotNil(err)
	if len(code_hash) == 0 {
		return
	}
	ret = self.db.GetCode(bin.HashView(code_hash))
	return
}
