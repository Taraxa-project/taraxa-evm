package trie

import "github.com/Taraxa-project/taraxa-evm/crypto"

type KeyHashingStorageStrategy byte

func (KeyHashingStorageStrategy) MapKey(key []byte) (mpt_key, flat_key []byte, err error) {
	return crypto.Keccak256(key), nil, nil
}
