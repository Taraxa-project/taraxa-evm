package main

import "github.com/Taraxa-project/taraxa-evm/core/state"

func Flush(statedb *state.StateDB) {
	statedb.Commit(false)
	trieDB := statedb.Database().TrieDB()
	for _, node := range trieDB.Nodes() {
		trieDB.Commit(node, true)
	}
}
