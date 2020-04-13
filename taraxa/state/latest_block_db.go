package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
)

type PendingBlockDB struct{ BlockDB }

func (self *PendingBlockDB) GetAccount(addr *common.Address) (ret Account, present bool) {
	enc_storage := self.db.GetMainTrieValueLatest(util.Hash(addr[:]))
	if present = len(enc_storage) != 0; present {
		ret.I_from_storage_encoding(enc_storage)
	}
	return
}

func (self *PendingBlockDB) GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int {
	if enc_storage := self.db.GetAccountTrieValueLatest(addr, util.Hash(key[:])); len(enc_storage) != 0 {
		_, val, _, err := rlp.Split(enc_storage)
		util.PanicIfNotNil(err)
		return new(big.Int).SetBytes(val)
	}
	return common.Big0
}
