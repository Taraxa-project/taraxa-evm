package misc

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"sync/atomic"
)

func DumpStateRocksdb(db_path_source, db_path_dest, root_str string) {
	root := common.HexToHash(root_str)
	rocksdb_source, err0 := (&rocksdb.Factory{
		File:                  db_path_source,
		ReadOnly:              true,
		MaxOpenFiles:          1000 * 4,
		Parallelism:           concurrent.NUM_CPU,
		MaxFileOpeningThreads: &concurrent.NUM_CPU,
		//BlockCacheSize:        1024 * 20,
		//BloomFilterCapacity:   20,
		OptimizeForPointLookup: func() *uint64 {
			ret := new(uint64)
			*ret = 1024 * 20
			return ret
		}(),
		UseDirectReads: true,
	}).NewInstance()
	util.PanicIfPresent(err0)
	//db_dest, err343 := (&rocksdb.Factory{
	//	File:                  db_path_dest,
	//	MaxOpenFiles:          1000 * 4,
	//	Parallelism:           concurrent.NUM_CPU,
	//	MaxFileOpeningThreads: &concurrent.NUM_CPU,
	//	//BlockCacheSize:        1024 * 1024 * 1024 * 5,
	//	//WriteBufferSize:       512 * 1024 * 1024,
	//	//BloomFilterCapacity:   10,
	//	//MergeOperartor:        rocksdb.NeverOverwrite,
	//}).NewInstance()
	//util.PanicIfPresent(err343)
	db_source := state.NewDatabaseWithCache(rocksdb_source, 1024*30)
	acc_trie_source, err1 := db_source.OpenTrie(root)
	util.PanicIfPresent(err1)
	//trie_db_dest := trie.NewDatabaseWithCache(db_dest, 1024*4)
	acc_cnt := new(int32)
	err2 := acc_trie_source.VisitLeaves(func(key, value []byte, parent_hash common.Hash) error {
		acc := new(state.Account)
		if err := rlp.DecodeBytes(value, acc); err != nil {
			return err
		}
		//storage_trie, err1 := db_source.OpenTrie(acc.Root)
		//util.PanicIfPresent(err1)
		//storage_trie.VisitLeaves(func(key, value []byte, parent_hash common.Hash) error {
		//	db_dest.Put(key, value)
		//})
		_, err := json.Marshal(acc)
		if err != nil {
			return err
		}
		value_from_db, err2 := rocksdb_source.Get(key)
		util.PanicIfPresent(err2)
		for _, c := range value {
			fmt.Print(c)
		}
		fmt.Println()
		for _, c := range value_from_db {
			fmt.Print(c)
		}
		fmt.Println()
		//util.Assert(bytes.Equal(value, value_from_db))
		//err := db_dest.Put(key, value)
		//util.PanicIfPresent(err)
		//fmt.Println(common.BytesToAddress(key).Hex(), string(acc_json_bytes))
		fmt.Println(atomic.AddInt32(acc_cnt, 1))
		return nil
	})
	util.PanicIfPresent(err2)
	//state_lock := new(sync.Mutex)
	//running_count := new(int32)
	//scheduled_count := int32(0)
	//for acc_itr := trie.NewIterator(acc_trie_source.NodeIterator(nil)); acc_itr.Next(); {
	//	var acc state.Account
	//	err := rlp.DecodeBytes(acc_itr.Value, &acc)
	//	util.PanicIfPresent(err)
	//	addr := common.BytesToAddress(acc_trie_source.GetKey(acc_itr.Key))
	//	addrHash := crypto.Keccak256Hash(addr[:])
	//	var code []byte
	//	if !bytes.Equal(acc.CodeHash, emptyCodeHash) {
	//		var err error
	//		code, err = db_source.ContractCode(addrHash, common.BytesToHash(acc.CodeHash))
	//		util.PanicIfPresent(err)
	//	}
	//	for atomic.LoadInt32(running_count) > int32(concurrent.NUM_CPU*1) {
	//		//runtime.Gosched()
	//		time.Sleep(time.Second * 5)
	//	}
	//	atomic.AddInt32(running_count, 1)
	//	go func() {
	//		defer atomic.AddInt32(running_count, -1)
	//		storage_trie, err1 := db_source.OpenStorageTrie(addrHash, root)
	//		util.PanicIfPresent(err1)
	//		storage := make(map[common.Hash]common.Hash)
	//		for storage_itr := trie.NewIterator(storage_trie.NodeIterator(nil)); storage_itr.Next(); {
	//			//fmt.Println("storage", addr.Hex(), common.Bytes2Hex(storage_itr.Key))
	//			_, content, _, err := rlp.Split(storage_itr.Value)
	//			util.PanicIfPresent(err)
	//			storage[common.BytesToHash(storage_trie.GetKey(storage_itr.Key))] = common.BytesToHash(content)
	//		}
	//		defer concurrent.LockUnlock(state_lock)()
	//		state_dest.SetBalance(addr, acc.Balance)
	//		state_dest.SetNonce(addr, acc.Nonce)
	//		state_dest.SetCode(addr, code)
	//		state_dest.SetStorage(addr, storage)
	//	}()
	//	scheduled_count++
	//	fmt.Println("scheduled", scheduled_count)
	//}
	//for atomic.LoadInt32(running_count) != 0 {
	//	runtime.Gosched()
	//}
	//eth_db := eth_state.NewDatabase(&dbAdapter{db_dest})
	//eth_root := eth_common.Hash(root)
	//tr, err3434 := eth_db.OpenTrie(eth_root)
	//util.PanicIfPresent(err3434)
	//util.Assert(tr.Hash() == eth_root)
	//acc_cnt := 0
	//fmt.Println("foo")
	//for acc_itr := eth_trie.NewIterator(tr.NodeIterator(nil)); acc_itr.Next(); {
	//	acc_cnt++
	//	fmt.Println(acc_cnt, acc_itr)
	//}
}
