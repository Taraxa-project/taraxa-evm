package trie

type Database interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
}
