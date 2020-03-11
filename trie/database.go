package trie

type Database interface {
	PutAsync(key, value []byte)
	GetCommitted(key []byte) ([]byte, error)
}
