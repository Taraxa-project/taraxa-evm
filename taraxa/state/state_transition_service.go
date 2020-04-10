package state

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"math/big"
)

type StateTransitionService struct {
	last_blk              BlockState
	get_block_hash        vm.GetHashFunc
	chain_cfg             params.ChainConfig
	opts_exec             vm.ExecutionOptions
	disable_block_rewards bool
	main_tr_w             trie.TrieWriter
	main_tr_w_executor    util.SingleThreadExecutor
	util.InitFlag
}

func (self *StateTransitionService) I(
	db DB,
	last_blk_num types.BlockNum,
	get_block_hash vm.GetHashFunc,
	chain_config params.ChainConfig,
	opts_exec vm.ExecutionOptions,
	disable_block_rewards bool,
	last_root_hash *common.Hash,
	main_trie_writer_opts trie.TrieWriterOpts,
) *StateTransitionService {
	self.InitOnce()
	self.last_blk = BlockState{db, last_blk_num}
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_config
	self.disable_block_rewards = disable_block_rewards
	self.opts_exec = opts_exec
	self.main_tr_w.I(MainTrieSchema{}, main_trie_writer_opts, last_root_hash)
	return self
}

func (self *StateTransitionService) ApplyGenesis(accs core.GenesisAlloc) (ret common.Hash) {
	self.main_tr_w.SetIO(nil, &MainTrieOutput{self.last_blk})
	for addr, acc := range accs {
		trie_acc := Account{nonce: acc.Nonce, balance: acc.Balance, code_size: uint64(len(acc.Code))}
		if trie_acc.code_size != 0 {
			code_hash := util.Hash(acc.Code)
			trie_acc.code_hash = code_hash
			self.last_blk.db.PutCode(code_hash, acc.Code)
		}
		if len(acc.Storage) != 0 {
			var acc_tr_w acc_tr_w
			acc_tr_w.I(nil).SetIO(nil, &AccountTrieOutput{self.last_blk, &addr})
			for k, v := range acc.Storage {
				bs := new(big.Int).SetBytes(v[:]).Bytes()
				assert.Holds(len(bs) != 0)
				acc_tr_w.Put(util.Hash(k[:]), trie.RawValue{bs, bs})
			}
			trie_acc.storage_root_hash = acc_tr_w.Commit()
		}
		self.main_tr_w.Put(util.Hash(addr[:]), &trie_acc)
	}
	ret = *self.main_tr_w.Commit()
	return
}

type StateTransitionParams = struct {
	Block              *vm.Block
	Uncles             []ethash.BlockNumAndCoinbase
	Transactions       []vm.Transaction
	ConcurrentSchedule ConcurrentSchedule
}
type StateTransitionResult = struct {
	StateRoot        common.Hash
	ExecutionResults []vm.ExecutionResult
}

func (self *StateTransitionService) TransitionState(param StateTransitionParams) (ret StateTransitionResult) {
	ret.ExecutionResults = make([]vm.ExecutionResult, len(param.Transactions))
	next_blk := BlockState{self.last_blk.db, param.Block.Number}
	self.main_tr_w.SetIO(&MainTrieInput{self.last_blk}, &MainTrieOutput{next_blk})
	pending_accounts := make(map[common.Address]*pending_account, util.CeilPow2(len(param.Transactions)*2))
	evm_state_sink := EVMStateOutput{
		OnAccountChanged: func(addr common.Address, change AccountChange) {
			pending_acc := pending_accounts[addr]
			if pending_acc == nil {
				pending_acc = new(pending_account)
				pending_accounts[addr] = pending_acc
			}
			pending_acc.executor.Do(func() {
				pending_acc.acc = change.Account
				if change.code_dirty {
					self.last_blk.db.PutCode(change.code_hash, change.code)
				}
				if len(change.storage_dirty) == 0 {
					return
				}
				if pending_acc.trie_w.IsZero() {
					pending_acc.trie_w.I(pending_acc.acc.storage_root_hash).
						SetIO(&AccountTrieInput{self.last_blk, &addr}, &AccountTrieOutput{next_blk, &addr})
				}
				for k, v := range change.storage_dirty {
					if h := util.HashOnStack(k[:]); v.Sign() == 0 {
						pending_acc.trie_w.Delete(&h)
					} else {
						bs := v.Bytes()
						pending_acc.trie_w.Put(&h, trie.RawValue{bs, bs})
					}
				}
			})
			self.main_tr_w_executor.Do(func() {
				self.main_tr_w.Put(util.Hash(addr[:]), pending_acc)
			})
		},
		OnAccountDeleted: func(addr common.Address) {
			pending_accounts[addr] = nil
			self.main_tr_w_executor.Do(func() {
				self.main_tr_w.Delete(util.Hash(addr[:]))
			})
		},
	}
	state := NewEVMState(&self.last_blk, EvmStateOpts{
		AccountCacheSize:      len(pending_accounts) * 2,
		DirtyAccountCacheSize: len(pending_accounts),
	})
	rules := self.chain_cfg.Rules(param.Block.Number)
	evm_cfg := vm.NewEVMConfig(self.get_block_hash, param.Block, rules, self.opts_exec)
	if rules.IsDAOFork {
		misc.ApplyDAOHardFork(&state)
		state.Commit(rules.IsEIP158, evm_state_sink)
	}
	for i, cnt := TxIndex(0), TxIndex(len(param.Transactions)); i < cnt; i++ {
		ret.ExecutionResults[i] = vm.Main(&evm_cfg, &state, &param.Transactions[i])
		state.Commit(rules.IsEIP158, evm_state_sink)
	}
	if !self.disable_block_rewards {
		ethash.AccumulateRewards(
			rules,
			ethash.BlockNumAndCoinbase{param.Block.Number, param.Block.Author},
			param.Uncles,
			state.AddBalance)
		state.Commit(rules.IsEIP158, evm_state_sink)
	}
	for _, pending_acc := range pending_accounts {
		pending_acc.executor.Do(func() {
			if !pending_acc.trie_w.IsZero() {
				pending_acc.acc.storage_root_hash = pending_acc.trie_w.Commit()
			}
			pending_acc.enc_storage, pending_acc.enc_hash = pending_acc.acc.EncodeForTrie()
		})
	}
	self.main_tr_w_executor.Do(func() {
		if h := self.main_tr_w.Commit(); h != nil {
			ret.StateRoot = *h
		} else {
			ret.StateRoot = empty_rlp_list_hash
		}
	})
	self.main_tr_w_executor.Join()
	self.last_blk = next_blk
	self.main_tr_w = trie.TrieWriter{}
	self.main_tr_w.I(MainTrieSchema{}, trie.TrieWriterOpts{}, &ret.StateRoot)
	return
}

type pending_account struct {
	acc         Account
	trie_w      acc_tr_w
	executor    util.SingleThreadExecutor
	enc_storage []byte
	enc_hash    []byte
}

func (self *pending_account) EncodeForTrie() (r0, r1 []byte) {
	self.executor.Join()
	r0, r1 = self.enc_storage, self.enc_hash
	return
}

type acc_tr_w struct{ trie.TrieWriter }

func (self *acc_tr_w) I(root_hash *common.Hash) *acc_tr_w {
	self.TrieWriter.I(AccountTrieSchema{}, trie.TrieWriterOpts{}, root_hash)
	return self
}
