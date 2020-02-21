package trie

import "github.com/Taraxa-project/taraxa-evm/ethdb"

type Database interface {
	ethdb.Putter
	ethdb.Getter
}