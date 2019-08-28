package misc

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"github.com/Taraxa-project/taraxa-evm/trie"
)

func DumpStateRocksdb(db_path_source, db_path_dest, root_str string) {
	root := common.HexToHash(root_str)
	rocksdb_source, err0 := (&rocksdb.Factory{
		File:                  db_path_source,
		ReadOnly:              true,
		MaxOpenFiles:          1000 * 4,
		Parallelism:           concurrent.NUM_CPU,
		MaxFileOpeningThreads: &concurrent.NUM_CPU,
		BlockCacheSize:        1024 * 20,
		BloomFilterCapacity:   20,
		//OptimizeForPointLookup: func() *uint64 {
		//	ret := new(uint64)
		//	*ret = 1024 * 20
		//	return ret
		//}(),
		UseDirectReads: true,
	}).NewInstance()
	util.PanicIfPresent(err0)
	db_dest, err343 := (&rocksdb.Factory{
		File:                  db_path_dest,
		MaxOpenFiles:          1000 * 2,
		Parallelism:           concurrent.NUM_CPU,
		MaxFileOpeningThreads: &concurrent.NUM_CPU,
		BlockCacheSize:        1024 * 5,
		BloomFilterCapacity:   10,
	}).NewInstance()
	util.PanicIfPresent(err343)
	db_source := state.NewDatabaseWithCache(rocksdb_source, 1024*30)
	acc_trie_source, err1 := db_source.OpenTrie(root)
	util.PanicIfPresent(err1)
	trie_db_dest := trie.NewDatabaseWithCache(db_dest, 1024*4)
	root_dest, err2 := acc_trie_source.Dump(trie_db_dest)
	util.PanicIfPresent(err2)
	fmt.Println(root.Hex(), root_dest.Hex())
	util.Assert(root == root_dest)
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
	err3 := trie_db_dest.Commit(root, false)
	util.PanicIfPresent(err3)
	tr, err4 := trie.NewSecure(root_dest, trie_db_dest, 0)
	util.PanicIfPresent(err4)
	util.Assert(tr.Hash() == root_dest)
}
