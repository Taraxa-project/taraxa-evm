package main

//#include "common.h"
//#include "state.h"
//#include <rocksdb/c.h>
import "C"
import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_concurrent_schedule"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
	"math/big"
	"sync"
	"unsafe"
)

type state_API struct {
	db             state_db_rocksdb.DB
	get_blk_hash_C C.taraxa_evm_GetBlockHash
	state.API
}

func (self *state_API) blk_hash(num types.BlockNum) *big.Int {
	hash_c, err := C.taraxa_evm_GetBlockHashApply(self.get_blk_hash_C, C.uint64_t(num))
	util.PanicIfNotNil(err)
	return new(big.Int).SetBytes(bin.AnyBytes2(unsafe.Pointer(&hash_c.Val), common.HashLength))
}

//export taraxa_evm_state_API_New
func taraxa_evm_state_API_New(
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) C.taraxa_evm_state_API_ptr {
	defer handle_err(cb_err)
	var params struct {
		RocksDBPtr                 uintptr
		RocksDBColumnFamilyHandles [state_db_rocksdb.COL_COUNT]uintptr
		GetBlockHash               uintptr
		ChainConfig                state_common.ChainConfig
		CurrBlkNum                 types.BlockNum
		CurrStateRoot              common.Hash
		StateTransitionCacheOpts   state_transition.CacheOpts
	}
	fmt.Println("1")
	dec_rlp(params_enc, &params)
	fmt.Println("11")
	self := new(state_API)
	fmt.Println("111")
	rocksdb := gorocksdb.NewDBFromNative(unsafe.Pointer(params.RocksDBPtr))
	fmt.Println("1111")
	var columns state_db_rocksdb.Columns
	fmt.Println("11111")
	for i, ptr := range params.RocksDBColumnFamilyHandles {
		fmt.Println("111111")
		columns[i] = gorocksdb.NewNativeColumnFamilyHandle1(unsafe.Pointer(ptr))
		fmt.Println("1111111")
	}
	fmt.Println("11111111")
	self.db.Init(rocksdb, columns)
	fmt.Println("1111111111")
	self.get_blk_hash_C = *(*C.taraxa_evm_GetBlockHash)(unsafe.Pointer(params.GetBlockHash))
	fmt.Println("11111111111")
	self.API.Init(&self.db, self.blk_hash,
		params.ChainConfig, params.CurrBlkNum, params.CurrStateRoot, params.StateTransitionCacheOpts)
	fmt.Println("111111111111")
	defer util.LockUnlock(&state_API_alloc_mu)()
	fmt.Println("1111111111111")
	lastpos := len(state_API_available_ptrs) - 1
	fmt.Println("11111111111111")
	assert.Holds(lastpos >= 0)
	fmt.Println("111111111111111")
	ptr := state_API_available_ptrs[lastpos]
	fmt.Println("1111111111111111")
	state_API_available_ptrs = state_API_available_ptrs[:lastpos]
	fmt.Println("11111111111111111")
	state_API_instances[ptr] = self
	fmt.Println("111111111111111111")
	return C.taraxa_evm_state_API_ptr(ptr)
}

//export taraxa_evm_state_API_Free
func taraxa_evm_state_API_Free(
	ptr C.taraxa_evm_state_API_ptr,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	defer util.LockUnlock(&state_API_alloc_mu)()
	assert.Holds(state_API_instances[ptr] == nil)
	state_API_instances[ptr], state_API_available_ptrs = nil, append(state_API_available_ptrs, state_API_ptr(ptr))
}

//export taraxa_evm_state_API_NotifyStateTransitionCommitted
func taraxa_evm_state_API_NotifyStateTransitionCommitted(
	ptr C.taraxa_evm_state_API_ptr,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	self := state_API_instances[state_API_ptr(ptr)]
	self.db.Refresh()
}

//export taraxa_evm_state_API_Historical_Prove
func taraxa_evm_state_API_Historical_Prove(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum    types.BlockNum
		StateRoot common.Hash
		Addr      common.Address
		Keys      []common.Hash
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	ret := self.Historical.AtBlock(params.BlkNum).Prove(&params.StateRoot, &params.Addr, params.Keys...)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_API_Historical_GetAccount
func taraxa_evm_state_API_Historical_GetAccount(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	ret := self.Historical.AtBlock(params.BlkNum).GetAccountRaw(&params.Addr)
	call_bytes_cb(ret, cb)
}

//export taraxa_evm_state_API_Historical_GetAccountStorage
func taraxa_evm_state_API_Historical_GetAccountStorage(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
		Key    common.Hash
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	ret := self.Historical.AtBlock(params.BlkNum).GetAccountStorageRaw(&params.Addr, &params.Key)
	call_bytes_cb(ret, cb)
}

//export taraxa_evm_state_API_Historical_GetCodeByAddress
func taraxa_evm_state_API_Historical_GetCodeByAddress(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	ret := self.Historical.AtBlock(params.BlkNum).GetCodeByAddress(&params.Addr)
	call_bytes_cb(ret, cb)
}

//export taraxa_evm_state_API_DryRunner_Apply
func taraxa_evm_state_API_DryRunner_Apply(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Blk    vm.BlockWithoutNumber
		Trx    vm.Transaction
		Opts   *vm.ExecutionOptions
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	evm_blk := vm.Block{params.BlkNum, params.Blk}
	ret := self.DryRunner.Apply(&evm_blk, &params.Trx, params.Opts)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_API_StateTransition_ApplyAccounts
func taraxa_evm_state_API_StateTransition_ApplyAccounts(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BatchPtr   uintptr
		AccountMap state_transition.AccountMap
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	self.db.TransactionBegin(gorocksdb.NewNativeWriteBatch1(unsafe.Pointer(params.BatchPtr)))
	defer self.db.TransactionEnd()
	ret := self.StateTransition.ApplyAccounts(params.AccountMap)
	call_bytes_cb(bin.AnyBytes2(unsafe.Pointer(&ret), common.HashLength), cb)
}

//export taraxa_evm_state_API_StateTransition_ApplyBlock
func taraxa_evm_state_API_StateTransition_ApplyBlock(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BatchPtr           uintptr
		EVMBlock           vm.BlockWithoutNumber
		Transactions       []vm.Transaction
		Uncles             []state_transition.UncleBlock
		ConcurrentSchedule state_concurrent_schedule.ConcurrentSchedule
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	self.db.TransactionBegin(gorocksdb.NewNativeWriteBatch1(unsafe.Pointer(params.BatchPtr)))
	defer self.db.TransactionEnd()
	ret := self.StateTransition.ApplyBlock(
		&params.EVMBlock, params.Transactions, params.Uncles, params.ConcurrentSchedule)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_API_ConcurrentScheduleGeneration_Begin
func taraxa_evm_state_API_ConcurrentScheduleGeneration_Begin(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		Block vm.BlockWithoutNumber
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	self.ConcurrentScheduleGeneration.Begin(&params.Block)
}

//export taraxa_evm_state_API_ConcurrentScheduleGeneration_SubmitTransactions
func taraxa_evm_state_API_ConcurrentScheduleGeneration_SubmitTransactions(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		Transactions []state_concurrent_schedule.TransactionWithHash
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	self.ConcurrentScheduleGeneration.SubmitTransactions(params.Transactions...)
}

//export taraxa_evm_state_API_ConcurrentScheduleGeneration_Commit
func taraxa_evm_state_API_ConcurrentScheduleGeneration_Commit(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		TransactionHashes []common.Hash
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	ret := self.ConcurrentScheduleGeneration.Commit(params.TransactionHashes...)
	enc_rlp(&ret, cb)
}

type state_API_ptr = byte

const state_API_max_instances = ^state_API_ptr(0)

var state_API_alloc_mu sync.Mutex
var state_API_instances [state_API_max_instances]*state_API
var state_API_available_ptrs = func() (ret []state_API_ptr) {
	ret = make([]state_API_ptr, state_API_max_instances)
	for i := state_API_ptr(0); i < state_API_max_instances; i++ {
		ret[i] = i
	}
	return
}()