package misc

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/big"
	"sync"
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
	rocksdb_dest, err343 := (&rocksdb.Factory{
		File:                  db_path_dest,
		MaxOpenFiles:          1000 * 4,
		Parallelism:           concurrent.NUM_CPU,
		MaxFileOpeningThreads: &concurrent.NUM_CPU,
		//BlockCacheSize:        1024 * 1024 * 1024 * 5,
		//WriteBufferSize:       512 * 1024 * 1024,
		//BloomFilterCapacity:   10,
		//MergeOperartor:        rocksdb.NeverOverwrite,
	}).NewInstance()
	util.PanicIfPresent(err343)
	db_source := state.NewDatabaseWithCache(rocksdb_source, 1024*30)
	acc_trie_source, err1 := db_source.OpenTrie(root)
	util.PanicIfPresent(err1)
	db_dest := state.NewDatabaseWithCache(rocksdb_dest, 1024*10)
	state_db_dest, err43 := state.New(common.Hash{}, db_dest)
	util.PanicIfPresent(err43)
	state_db_mu := new(sync.Mutex)
	acc_cnt := new(uint32)
	err2 := acc_trie_source.VisitLeaves(func(key, value []byte, parent_hash common.Hash) error {
		addr := common.BytesToAddress(key)
		acc := new(state.Account)
		if err := rlp.DecodeBytes(value, acc); err != nil {
			return err
		}
		addrHash := crypto.Keccak256Hash(addr[:])
		code, err23 := db_dest.ContractCode(addrHash, common.BytesToHash(acc.CodeHash))
		util.PanicIfPresent(err23)
		storage, storage_mu := make(map[common.Hash]common.Hash), new(sync.Mutex)
		storage_trie, err1 := db_source.OpenTrie(acc.Root)
		util.PanicIfPresent(err1)
		storage_trie.VisitLeaves(func(key, value []byte, parent_hash common.Hash) error {
			defer concurrent.LockUnlock(storage_mu)()
			storage[common.BytesToHash(key)] = common.BytesToHash(value)
			return nil
		})
		var intermediate_root *common.Hash
		concurrent.WithLock(state_db_mu, func() {
			state_db_dest.SetBalance(addr, new(big.Int).Set(acc.Balance))
			state_db_dest.SetNonce(addr, acc.Nonce)
			state_db_dest.SetCode(addr, code)
			for k, v := range storage {
				state_db_dest.SetState(addr, k, v)
			}
			root, err133 := state_db_dest.Commit(false)
			util.PanicIfPresent(err133)
			*intermediate_root = root
		})
		err13443 := db_dest.TrieDB().Commit(*intermediate_root, false)
		util.PanicIfPresent(err13443)
		fmt.Println(atomic.AddUint32(acc_cnt, 1))
		return nil
	})
	util.PanicIfPresent(err2)
}
