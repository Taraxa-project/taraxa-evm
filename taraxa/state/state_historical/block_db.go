package state_historical

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/util_rlp"
	"math/big"
)

type BlockDB struct {
	db      state_common.DB
	blk_num types.BlockNum
}

func (self BlockDB) GetCode(code_hash *common.Hash) (ret []byte) {
	return self.db.GetCode(code_hash)
}

func (self BlockDB) GetAccount(addr *common.Address) (ret state_common.Account, present bool) {
	enc_storage := self.GetAccountRaw(addr)
	if present = len(enc_storage) != 0; present {
		state_common.DecodeAccount(&ret, enc_storage)
	}
	return
}

func (self BlockDB) GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int {
	if enc_storage := self.GetAccountStorageRaw(addr, key); len(enc_storage) != 0 {
		return new(big.Int).SetBytes(enc_storage)
	}
	return common.Big0
}

type Proof struct {
	AccountProof  trie.Proof
	StorageProofs []trie.Proof
}

func (self BlockDB) Prove(state_root *common.Hash, addr *common.Address, keys ...common.Hash) (ret Proof) {
	ret.AccountProof = trie.Reader{main_trie_db{BlockDB: self}}.Prove(state_root, util.Hash(addr[:]))
	if len(ret.AccountProof.Value) == 0 || len(keys) == 0 {
		return
	}
	ret.StorageProofs = make([]trie.Proof, len(keys))
	storage_root := util_rlp.RLPListAt(ret.AccountProof.Value, 2)
	if len(storage_root) == 0 {
		return
	}
	acc_tr_reader := trie.Reader{account_trie_db{BlockDB: self, addr: addr}}
	storage_root_h := bin.HashView(storage_root)
	for i := 0; i < len(keys); i++ {
		ret.StorageProofs[i] = acc_tr_reader.Prove(storage_root_h, util.Hash(keys[i][:]))
	}
	return
}

func (self BlockDB) GetAccountRaw(addr *common.Address) []byte {
	return self.db.GetMainTrieValue(self.blk_num, util.Hash(addr[:]))
}

func (self BlockDB) GetAccountStorageRaw(addr *common.Address, key *common.Hash) (ret []byte) {
	key_hash := util.HashOnStack(key[:])
	if ret = self.db.GetAccountTrieValue(self.blk_num, addr, &key_hash); len(ret) != 0 {
		_, ret, _ = rlp.MustSplit(ret)
	}
	return
}

func (self BlockDB) GetCodeByAddress(addr *common.Address) (ret []byte) {
	if acc := self.GetAccountRaw(addr); len(acc) != 0 {
		if code_hash := util_rlp.RLPListAt(acc, 3); len(code_hash) != 0 {
			ret = self.db.GetCode(bin.HashView(code_hash))
		}
	}
	return
}
