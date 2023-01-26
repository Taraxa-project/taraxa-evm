package main

//#include "common.h"
//#include "state.h"
//#include <rocksdb/c.h>
import "C"
import (
	"fmt"
	"math/big"
	"sync"
	"unsafe"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/chain_config"
	dpos "github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos/precompiled"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/rewards_stats"
	"github.com/holiman/uint256"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
)

type state_API struct {
	state.API
	db             state_db_rocksdb.DB
	get_blk_hash_C C.taraxa_evm_GetBlockHash
}

func (self *state_API) blk_hash(num types.BlockNum) *big.Int {
	hash_c, err := C.taraxa_evm_GetBlockHashApply(self.get_blk_hash_C, C.uint64_t(num))
	util.PanicIfNotNil(err)
	return new(big.Int).SetBytes(bin.AnyBytes2(unsafe.Pointer(&hash_c.Val), common.HashLength))
}

//export taraxa_evm_state_api_new
func taraxa_evm_state_api_new(
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) C.taraxa_evm_state_API_ptr {
	defer handle_err(cb_err)
	var params struct {
		GetBlockHash uintptr
		ChainConfig  *chain_config.ChainConfig
		Opts         state.APIOpts
		OptsDB       state_db_rocksdb.Opts
	}
	dec_rlp(params_enc, &params)
	self := new(state_API)
	self.db.Init(params.OptsDB)
	self.get_blk_hash_C = *(*C.taraxa_evm_GetBlockHash)(unsafe.Pointer(params.GetBlockHash))
	self.Init(&self.db, self.blk_hash, params.ChainConfig, params.Opts)

	defer util.LockUnlock(&state_API_alloc_mu)()
	lastpos := len(state_API_available_ptrs) - 1
	asserts.Holds(lastpos >= 0)
	ptr := state_API_available_ptrs[lastpos]
	asserts.Holds(state_API_instances[ptr] == nil)
	state_API_available_ptrs = state_API_available_ptrs[:lastpos]
	state_API_instances[ptr] = self
	return C.taraxa_evm_state_API_ptr(ptr)
}

//export taraxa_evm_state_api_free
func taraxa_evm_state_api_free(
	ptr C.taraxa_evm_state_API_ptr,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	self := state_API_instances[ptr]
	self.Close()
	self.db.Close()
	defer util.LockUnlock(&state_API_alloc_mu)()
	state_API_instances[ptr], state_API_available_ptrs = nil, append(state_API_available_ptrs, state_API_ptr(ptr))
}

//export taraxa_evm_state_api_get_last_committed_state_descriptor
func taraxa_evm_state_api_get_last_committed_state_descriptor(
	ptr C.taraxa_evm_state_API_ptr,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	ret := state_API_instances[ptr].GetCommittedStateDescriptor()
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_api_prove
func taraxa_evm_state_api_prove(
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
	ret := state_API_instances[ptr].ReadBlock(params.BlkNum).Prove(&params.StateRoot, &params.Addr, params.Keys...)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_api_update_state_config
func taraxa_evm_state_api_update_state_config(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		ChainConfig chain_config.ChainConfig
	}
	dec_rlp(params_enc, &params)
	self := state_API_instances[ptr]
	self.UpdateConfig(&params.ChainConfig)
}

//export taraxa_evm_state_api_get_account
func taraxa_evm_state_api_get_account(
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
	state_API_instances[ptr].ReadBlock(params.BlkNum).GetRawAccount(&params.Addr, func(bytes []byte) {
		call_bytes_cb(bytes, cb)
	})
}

//export taraxa_evm_state_api_get_account_storage
func taraxa_evm_state_api_get_account_storage(
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
	state_API_instances[ptr].ReadBlock(params.BlkNum).GetAccountStorage(&params.Addr, &params.Key, func(bytes []byte) {
		call_bytes_cb(bytes, cb)
	})
}

//export taraxa_evm_state_api_get_code_by_address
func taraxa_evm_state_api_get_code_by_address(
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
	ret := state_API_instances[ptr].ReadBlock(params.BlkNum).GetCodeByAddress(&params.Addr)
	call_bytes_cb(ret, cb)
}

//export taraxa_evm_state_api_dry_run_transaction
func taraxa_evm_state_api_dry_run_transaction(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Blk    vm.BlockInfo
		Trx    vm.Transaction
	}
	dec_rlp(params_enc, &params)
	ret := state_API_instances[ptr].DryRunTransaction(&vm.Block{params.BlkNum, params.Blk}, &params.Trx)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_api_trace_transaction
func taraxa_evm_state_api_trace_transaction(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Blk    vm.BlockInfo
		Trx    vm.Transaction
		Params vm.TracingConfig
	}
	dec_rlp(params_enc, &params)
	ret := state_API_instances[ptr].TraceTransaction(&vm.Block{params.BlkNum, params.Blk}, &params.Trx, &params.Params)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_api_debug_transaction
func taraxa_evm_state_api_debug_transaction(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Blk    vm.BlockInfo
		Trx    vm.Transaction
	}
	dec_rlp(params_enc, &params)
	ret := state_API_instances[ptr].TraceTransaction(&vm.Block{params.BlkNum, params.Blk}, &params.Trx, nil)
	enc_rlp(&ret, cb)
}

//export taraxa_evm_state_api_transition_state
func taraxa_evm_state_api_transition_state(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		Blk           vm.BlockInfo
		Txs           []vm.Transaction
		TxsValidators []common.Address // Transactions stats: tx hash -> validator that included it as first in his block
		Uncles        []state_common.UncleBlock
		Rewards_stats rewards_stats.RewardsStats
	}
	dec_rlp(params_enc, &params)

	var retval struct {
		ExecutionResults []vm.ExecutionResult
		StateRoot        common.Hash
	}
	self := state_API_instances[ptr]
	st := self.GetStateTransition()

	disabled_contract_distribution := st.GetChainConfig().RewardsEnabled()
	if !disabled_contract_distribution && len(params.Txs) != len(params.TxsValidators) {
		errorString := fmt.Sprintf("Number of txs (%d) != number of txs validators (%d)", len(params.Txs), len(params.TxsValidators))
		panic(errorString)
	}

	// What rewards should be distributed to which accounts
	feesRewards := dpos.NewFeesRewards()

	st.BeginBlock(&params.Blk)

	for i := range params.Txs {
		tx := &params.Txs[i]
		txResult := st.ExecuteTransaction(tx)

		txFee := new(uint256.Int).SetUint64(txResult.GasUsed)
		g, _ := uint256.FromBig(tx.GasPrice)
		txFee.Mul(txFee, g)

		// Contract distribution is disabled - just add fee to the block author balance
		// TODO: once there is a stabilized version - remove this flag and use only dpos contract
		if disabled_contract_distribution {
			st.AddTxFeeToBalance(&params.Blk.Author, txFee)
		} else {
			// Reward dag block author, who included specified tx as first
			feesRewards.AddTrxFeeReward(params.TxsValidators[i], txFee)
		}

		retval.ExecutionResults = append(retval.ExecutionResults, txResult)
	}

	st.EndBlock(params.Uncles, &params.Rewards_stats, &feesRewards)
	retval.StateRoot = st.PrepareCommit()
	enc_rlp(&retval, cb)
}

//export taraxa_evm_state_api_transition_state_commit
func taraxa_evm_state_api_transition_state_commit(
	ptr C.taraxa_evm_state_API_ptr,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	state_API_instances[ptr].GetStateTransition().Commit()
}

//export taraxa_evm_state_api_dpos_is_eligible
func taraxa_evm_state_api_dpos_is_eligible(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) bool {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	return state_API_instances[ptr].DPOSReader(params.BlkNum).IsEligible(&params.Addr)
}

//export taraxa_evm_state_api_dpos_get_staking_balance
func taraxa_evm_state_api_dpos_get_staking_balance(
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
	call_bytes_cb(state_API_instances[ptr].DPOSReader(params.BlkNum).GetStakingBalance(&params.Addr).Bytes(), cb)
}

//export taraxa_evm_state_api_dpos_get_vrf_key
func taraxa_evm_state_api_dpos_get_vrf_key(
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
	call_bytes_cb(state_API_instances[ptr].DPOSReader(params.BlkNum).GetVrfKey(&params.Addr), cb)
}

//export taraxa_evm_state_api_dpos_total_amount_delegated
func taraxa_evm_state_api_dpos_total_amount_delegated(
	ptr C.taraxa_evm_state_API_ptr,
	blk_n uint64,
	cb C.taraxa_evm_BytesCallback,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	call_bytes_cb(state_API_instances[ptr].DPOSReader(blk_n).TotalAmountDelegated().Bytes(), cb)
}

//export taraxa_evm_state_api_dpos_eligible_vote_count
func taraxa_evm_state_api_dpos_eligible_vote_count(
	ptr C.taraxa_evm_state_API_ptr,
	blk_n uint64,
	cb_err C.taraxa_evm_BytesCallback,
) uint64 {
	defer handle_err(cb_err)
	return state_API_instances[ptr].DPOSReader(blk_n).EligibleVoteCount()
}

//export taraxa_evm_state_api_dpos_get_eligible_vote_count
func taraxa_evm_state_api_dpos_get_eligible_vote_count(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) uint64 {
	defer handle_err(cb_err)
	var params struct {
		BlkNum types.BlockNum
		Addr   common.Address
	}
	dec_rlp(params_enc, &params)
	return state_API_instances[ptr].DPOSReader(params.BlkNum).GetEligibleVoteCount(&params.Addr)
}

//export taraxa_evm_state_api_db_snapshot
func taraxa_evm_state_api_db_snapshot(
	ptr C.taraxa_evm_state_API_ptr,
	dir string,
	log_size_for_flush uint64,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	db := &state_API_instances[ptr].db
	util.PanicIfNotNil(db.Snapshot(dir, log_size_for_flush))
}

//export taraxa_evm_state_api_prune
func taraxa_evm_state_api_prune(
	ptr C.taraxa_evm_state_API_ptr,
	params_enc C.taraxa_evm_Bytes,
	cb_err C.taraxa_evm_BytesCallback,
) {
	defer handle_err(cb_err)
	var params struct {
		StateRootToKeep  common.Hash
		StateRootToPrune []common.Hash
		BlkNum           types.BlockNum
	}
	dec_rlp(params_enc, &params)
	state_API_instances[ptr].db.Prune(params.StateRootToKeep, params.StateRootToPrune, params.BlkNum)
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
