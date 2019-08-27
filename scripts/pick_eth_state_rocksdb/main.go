package main

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
	"os"
)

func main() {
	db_path_source, root_str := os.Args[1], os.Args[2]
	root := common.HexToHash(root_str)
	rocksdb_source, err0 := (&rocksdb.Factory{
		File:     db_path_source,
		ReadOnly: true,
	}).NewInstance()
	util.PanicIfPresent(err0)
	state_source, err1 := state.New(root, state.NewDatabase(&dbAdapter{rocksdb_source}))
	util.PanicIfPresent(err1)
	db_dest := ethdb.NewMemDatabase()
	state_dest, err2 := state.New(common.Hash{}, state.NewDatabase(&dbAdapter{db_dest}))
	util.PanicIfPresent(err2)
	dump := state_source.RawDump(false, false, false)
	for addr, acc := range dump.Accounts {
		state_dest.SetNonce(addr, acc.Nonce)
		balance, parsed := new(big.Int).SetString(acc.Balance, 10)
		util.Assert(parsed)
		state_dest.SetBalance(addr, balance)
		state_dest.SetCode(addr, common.Hex2Bytes(acc.Code))
		storage := make(map[common.Hash]common.Hash, len(acc.Storage))
		for loc, val := range acc.Storage {
			storage[loc] = common.HexToHash(val)
		}
		state_dest.SetStorage(addr, storage)
	}
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
