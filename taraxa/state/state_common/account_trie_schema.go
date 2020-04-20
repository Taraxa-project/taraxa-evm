package state_common

type AccountTrieSchema struct{}

func (AccountTrieSchema) ValueStorageToHashEncoding(enc_storage []byte) []byte { return enc_storage }
func (AccountTrieSchema) MaxValueSizeToStoreInTrie() int                       { return 8 }
