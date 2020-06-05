package state_common

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
)

type DB interface {
	GetCode(code_hash *common.Hash) []byte
	GetMainTrieNode(node_hash *common.Hash) []byte
	GetMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash) []byte
	GetMainTrieValueLatest(addr_hash *common.Hash) []byte
	GetAccountTrieNode(node_hash *common.Hash) []byte
	GetAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash) []byte
	GetAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash) []byte
	PutCode(code_hash *common.Hash, code []byte)
	DeleteCode(code_hash *common.Hash)
	PutMainTrieNode(node_hash *common.Hash, node []byte)
	PutMainTrieValue(block_num types.BlockNum, addr_hash *common.Hash, v []byte)
	PutMainTrieValueLatest(addr_hash *common.Hash, v []byte)
	DeleteMainTrieValueLatest(addr_hash *common.Hash)
	PutAccountTrieNode(node_hash *common.Hash, node []byte)
	PutAccountTrieValue(block_num types.BlockNum, addr *common.Address, key_hash *common.Hash, v []byte)
	PutAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash, v []byte)
	DeleteAccountTrieValueLatest(addr *common.Address, key_hash *common.Hash)
}
