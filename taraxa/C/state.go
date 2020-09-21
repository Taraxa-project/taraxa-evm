package main

//#include "common.h"
//#include "state.h"
//#include <rocksdb/c.h>
import "C"
import (
	"math/big"
	"sync"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_config"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_transition"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
)

type state_API struct {
	db                                state_db_rocksdb.DB
	get_blk_hash_C                    C.taraxa_evm_GetBlockHash
	params_StateTransition_ApplyBlock struct {
		BatchPtr     uintptr
		EVMBlock     vm.BlockWithoutNumber
		Transactions []vm.Transaction
		Uncles       []state_common.UncleBlock
	}
	rlp_buf_StateTransition_ApplyBlock []byte
	rlp_encoder_StateTransition_Apply  rlp.Encoder
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
		ChainConfig                state_config.ChainConfig
		CurrBlkNum                 types.BlockNum
		CurrStateRoot              common.Hash
		StateTransitionCacheOpts   state_transition.StateTransitionOpts
	}
	dec_rlp(params_enc, &params)
	self := new(state_API)
	rocksdb := gorocksdb.NewDBFromNative(unsafe.Pointer(params.RocksDBPtr))
	var columns state_db_rocksdb.Columns
	for i, ptr := range params.RocksDBColumnFamilyHandles {
		columns[i] = gorocksdb.NewNativeColumnFamilyHandle1(unsafe.Pointer(ptr))
	}
	self.db.Init(rocksdb, columns)
	self.get_blk_hash_C = *(*C.taraxa_evm_GetBlockHash)(unsafe.Pointer(params.GetBlockHash))
	self.API.Init(&self.db, self.blk_hash,
		params.ChainConfig, params.CurrBlkNum, &params.CurrStateRoot, params.StateTransitionCacheOpts)
	self.params_StateTransition_ApplyBlock.Transactions =
		make([]vm.Transaction, 0, params.StateTransitionCacheOpts.ExpectedMaxNumTrxPerBlock)
	self.rlp_buf_StateTransition_ApplyBlock =
		make([]byte, 0, params.StateTransitionCacheOpts.ExpectedMaxNumTrxPerBlock*1024)
	self.rlp_encoder_StateTransition_Apply.ResizeReset(
		cap(self.rlp_buf_StateTransition_ApplyBlock),
		int(params.StateTransitionCacheOpts.ExpectedMaxNumTrxPerBlock*128))
	defer util.LockUnlock(&state_API_alloc_mu)()
	lastpos := len(state_API_available_ptrs) - 1
	assert.Holds(lastpos >= 0)
	ptr := state_API_available_ptrs[lastpos]
	assert.Holds(state_API_instances[ptr] == nil)
	state_API_available_ptrs = state_API_available_ptrs[:lastpos]
	state_API_instances[ptr] = self
	return C.taraxa_evm_state_API_ptr(ptr)
}

//export taraxa_evm_state_API_Free
func taraxa_evm_state_API_Free(
	ptr C.taraxa_evm_state_API_ptr,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	defer util.LockUnlock(&state_API_alloc_mu)()
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
	blk, txn := self.Historical.ReadBlock(params.BlkNum)
	defer txn.NotifyDoneReading()
	ret := blk.Prove(&params.StateRoot, &params.Addr, params.Keys...)
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
	blk, txn := self.Historical.ReadBlock(params.BlkNum)
	defer txn.NotifyDoneReading()
	blk.GetRawAccount(&params.Addr, func(bytes []byte) {
		call_bytes_cb(bytes, cb)
	})
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
	blk, txn := self.Historical.ReadBlock(params.BlkNum)
	defer txn.NotifyDoneReading()
	blk.GetAccountStorage(&params.Addr, &params.Key, func(bytes []byte) {
		call_bytes_cb(bytes, cb)
	})
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
	blk, txn := self.Historical.ReadBlock(params.BlkNum)
	defer txn.NotifyDoneReading()
	ret := blk.GetCodeByAddress(&params.Addr)
	defer ret.Free()
	call_bytes_cb(ret.Value(), cb)
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
		Opts   *vm.ExecutionOptions `rlp:"nil"`
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	ret := self.DryRunner.Apply(params.BlkNum, &params.Blk, &params.Trx, params.Opts)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_API_StateTransition_GenesisInit
func taraxa_evm_state_API_StateTransition_GenesisInit(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BatchPtr uintptr
		Config   state_transition.GenesisConfig
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[state_API_ptr(ptr)]
	self.db.BatchBegin(gorocksdb.NewNativeWriteBatch1(unsafe.Pointer(params.BatchPtr)))
	defer self.db.BatchEnd()
	ret := self.StateTransition.GenesisInit(params.Config)
	call_bytes_cb(bin.AnyBytes2(unsafe.Pointer(&ret), common.HashLength), cb)
}

//export taraxa_evm_state_API_StateTransition_Apply
func taraxa_evm_state_API_StateTransition_Apply(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	self := state_API_instances[state_API_ptr(ptr)]
	params := &self.params_StateTransition_ApplyBlock
	params.Transactions = params.Transactions[:0]
	params.Uncles = params.Uncles[:0]
	dec_rlp(params_enc, params)
	self.db.BatchBegin(gorocksdb.NewNativeWriteBatch1(unsafe.Pointer(params.BatchPtr)))
	defer self.db.BatchEnd()
	self.StateTransition.BeginBlock(&params.EVMBlock)
	for i := range self.params_StateTransition_ApplyBlock.Transactions {
		self.StateTransition.SubmitTransaction(&self.params_StateTransition_ApplyBlock.Transactions[i])
	}
	self.StateTransition.EndBlock(self.params_StateTransition_ApplyBlock.Uncles)
	ret := self.StateTransition.CommitSync()
	self.rlp_encoder_StateTransition_Apply.Reset()
	self.rlp_encoder_StateTransition_Apply.AppendAny(ret)
	buf := self.rlp_buf_StateTransition_ApplyBlock[:0]
	self.rlp_encoder_StateTransition_Apply.FlushToBytes(-1, &buf)
	call_bytes_cb(buf, cb)
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
