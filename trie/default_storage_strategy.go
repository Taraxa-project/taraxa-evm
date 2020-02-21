package trie

type DefaultStorageStrategy byte

func (DefaultStorageStrategy) MapKey(key []byte) (mpt_key, flat_key []byte, err error) {
	return key, nil, nil
}
