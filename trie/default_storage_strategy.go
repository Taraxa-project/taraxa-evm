package trie

import "github.com/Taraxa-project/taraxa-evm/crypto"

type DefaultStorageStrategy byte

func (DefaultStorageStrategy) OriginKeyToMPTKey(key []byte) (mpt_key []byte, err error) {
	return crypto.Keccak256(key), nil
}

func (DefaultStorageStrategy) MPTKeyToFlat([]byte) (flat_key []byte, err error) {
	return nil, nil
}