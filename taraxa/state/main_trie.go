package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

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
		encoder.AppendString(empty_rlp_list_hash[:])
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

type MainTrieInputHistorical struct {
	MainTrieSchema
	*BlockDB
}

func (self MainTrieInputHistorical) GetValue(key *common.Hash) []byte {
	return self.db.GetMainTrieValue(self.blk_num, key)
}

func (self MainTrieInputHistorical) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetMainTrieNode(node_hash)
}

type MainTrieIOPending struct {
	MainTrieSchema
	*PendingBlockDB
}

func (self MainTrieIOPending) GetValue(key *common.Hash) []byte {
	return self.db.GetMainTrieValueLatest(key)
}

func (self MainTrieIOPending) GetNode(node_hash *common.Hash) []byte {
	return self.db.GetMainTrieNode(node_hash)
}

func (self MainTrieIOPending) PutValue(key *common.Hash, v []byte) {
	self.db.PutMainTrieValue(self.blk_num, key, v)
	self.db.PutMainTrieValueLatest(key, v)
}

func (self MainTrieIOPending) DeleteValue(key *common.Hash) {
	self.db.PutMainTrieValue(self.blk_num, key, nil)
	self.db.DeleteMainTrieValueLatest(key)
}

func (self MainTrieIOPending) PutNode(node_hash *common.Hash, node []byte) {
	self.db.PutMainTrieNode(node_hash, node)
}
