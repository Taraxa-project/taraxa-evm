package misc

import (
	"bytes"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
)

var emptyCodeHash = crypto.Keccak256(nil)

func DumpStateRocksdb(db_path_source, root_str string) {
	fmt.Println("foo")
	root := common.HexToHash(root_str)
	rocksdb_source, err0 := (&rocksdb.Factory{
		File:     db_path_source,
		ReadOnly: true,
	}).NewInstance()
	util.PanicIfPresent(err0)
	db_source := state.NewDatabase(&dbAdapter{rocksdb_source})
	acc_trie_source, err1 := db_source.OpenTrie(root)
	util.PanicIfPresent(err1)
	db_dest := ethdb.NewMemDatabase()
	state_dest, err2 := state.New(common.Hash{}, state.NewDatabase(&dbAdapter{db_dest}))
	util.PanicIfPresent(err2)
	fmt.Println("bar")
	for acc_itr := trie.NewIterator(acc_trie_source.NodeIterator(nil)); acc_itr.Next(); {
		var acc state.Account
		err := rlp.DecodeBytes(acc_itr.Value, &acc)
		util.PanicIfPresent(err)
		addr := common.BytesToAddress(acc_trie_source.GetKey(acc_itr.Key))
		addrHash := crypto.Keccak256Hash(addr[:])
		state_dest.SetBalance(addr, acc.Balance)
		state_dest.SetNonce(addr, acc.Nonce)
		var code []byte
		if !bytes.Equal(acc.CodeHash, emptyCodeHash) {
			var err error
			code, err = db_source.ContractCode(addrHash, common.BytesToHash(acc.CodeHash))
			util.PanicIfPresent(err)
		}
		state_dest.SetCode(addr, code)
		storage_trie, err1 := db_source.OpenStorageTrie(addrHash, root)
		util.PanicIfPresent(err1)
		for storage_itr := trie.NewIterator(storage_trie.NodeIterator(nil)); storage_itr.Next(); {
			_, content, _, err := rlp.Split(storage_itr.Value)
			util.PanicIfPresent(err)
			state_dest.SetState(
				addr,
				common.BytesToHash(storage_trie.GetKey(storage_itr.Key)),
				common.BytesToHash(content))
		}
		fmt.Println(addr.Hex())
	}
	fmt.Println("baz")
	root_dest, err3 := state_dest.Commit(false)
	util.PanicIfPresent(err3)
	util.Assert(root == root_dest)
	err4 := state_dest.Database().TrieDB().Commit(root_dest, false)
	util.PanicIfPresent(err4)
	for _, k := range db_dest.Keys() {
		v, err5 := db_dest.Get(k)
		util.PanicIfPresent(err5)
		fmt.Println(string(k), string(v))
	}
}
