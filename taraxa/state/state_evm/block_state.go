package state_evm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

type BlockState struct {
	state TransitionState
}

func (bs *BlockState) Init(opts Opts) {
	if opts.NumTransactionsToBuffer == 0 {
		opts.NumTransactionsToBuffer = 1
	}
	bs.state.accounts.Init(AccountMapOptions{opts.NumTransactionsToBuffer * 32, 4})
	bs.state.reverts_original = make([]func(), 0, 1024) // 8KB
	bs.state.reverts = bs.state.reverts_original
}

func GetBlockState(db state_db.DB, blk_n types.BlockNum, num_transactions int) *BlockState {
	bs := &BlockState{}
	bs.Init(Opts{NumTransactionsToBuffer: uint64(num_transactions)})
	bs.SetInput(state_db.GetBlockStateReader(db, blk_n))
	return bs
}

func (bs *BlockState) In() Input {
	return bs.state.in
}

func (bs *BlockState) SetInput(in Input) {
	bs.state.in = in
}

func (bs *BlockState) GetAccountConcrete(addr *common.Address) *Account {
	acc, was_present := bs.state.accounts.GetOrNew(addr)
	if was_present {
		return acc
	}
	acc.host = bs
	bs.state.in.GetAccount(addr, func(db_acc state_db.Account) {
		acc.AccountBody = &AccountBody{AccountChange: AccountChange{Account: db_acc}}
		acc.loaded_from_db = true
	})
	return acc
}

// CommitTransaction should do nothing as this state shouldn't be committed
func (bs *BlockState) CommitTransaction(db_writer Output) {
}

// Commit should do nothing as this state shouldn't be committed
func (bs *BlockState) Commit() {
}

func (bs *BlockState) GetAccount(addr *common.Address) vm.StateAccount {
	return bs.GetAccountConcrete(addr)
}

func (bs *BlockState) DeleteAccount(acc *Account) {
	bs.state.DeleteAccount(acc)
}

func (bs *BlockState) GetAccountStorage(addr *common.Address, key *common.Hash, cb func([]byte)) {
	bs.GetAccountConcrete(addr).GetRawState(key, cb)
}

func (bs *BlockState) GetAccountStorageFromDB(addr *common.Address, k *common.Hash, cb func([]byte)) {
	bs.In().GetAccountStorage(addr, k, cb)
}

func (bs *BlockState) AddLog(log vm.LogRecord) {
	bs.state.AddLog(log)
}

func (bs *BlockState) GetLogs() []vm.LogRecord {
	return bs.state.logs
}

func (bs *BlockState) AddRefund(gas uint64) {
	bs.state.AddRefund(gas)
}

func (bs *BlockState) SubRefund(gas uint64) {
	bs.state.SubRefund(gas)
}

func (bs *BlockState) GetRefund() uint64 {
	return bs.state.GetRefund()
}

func (bs *BlockState) Snapshot() int {
	return bs.state.Snapshot()
}

func (bs *BlockState) RevertToSnapshot(snapshot int) {
	bs.state.RevertToSnapshot(snapshot)
}

func (bs *BlockState) RegisterChange(revert func()) {
	bs.state.RegisterChange(revert)
}

func (bs *BlockState) initTransientState() {
	bs.state.initTransientState()
}

// SetTransientState sets transient storage for a given account. It
// adds the change to the journal so that it can be rolled back
// to its previous value if there is a revert.
func (bs *BlockState) SetTransientState(addr *common.Address, key, value common.Hash) {
	bs.state.SetTransientState(addr, key, value)
}

// setTransientState is a lower level setter for transient storage. It
// is called during a revert to prevent modifications to the journal.
func (bs *BlockState) setTransientState(addr *common.Address, key, value common.Hash) {
	bs.state.setTransientState(addr, key, value)
}

// GetTransientState gets transient storage for a given account.
func (bs *BlockState) GetTransientState(addr *common.Address, key common.Hash) common.Hash {
	return bs.state.GetTransientState(addr, key)
}
