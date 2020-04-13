package state

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/dbg"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"math/big"
)

type StateTransitionService struct {
	db                    PendingBlockDB
	get_block_hash        vm.GetHashFunc
	chain_cfg             params.ChainConfig
	opts_exec             vm.ExecutionOptions
	disable_block_rewards bool
	main_tr_w             trie.TrieWriter
	main_tr_w_executor    util.SingleThreadExecutor
}

func (self *StateTransitionService) Init(
	db DB,
	last_root_hash *common.Hash,
	main_trie_writer_opts trie.TrieWriterOpts,
	opts_exec vm.ExecutionOptions,
	disable_block_rewards bool,
	get_block_hash vm.GetHashFunc,
	chain_config params.ChainConfig,
) *StateTransitionService {
	self.db.db = db
	self.main_tr_w.Init(MainTrieIOPending{PendingBlockDB: &self.db}, last_root_hash, main_trie_writer_opts)
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_config
	self.disable_block_rewards = disable_block_rewards
	self.opts_exec = opts_exec
	return self
}

func (self *StateTransitionService) HashFully() *common.Hash {
	return self.main_tr_w.HashFully()
}

func (self *StateTransitionService) ApplyGenesis(accs core.GenesisAlloc) (ret common.Hash) {
	for addr, acc := range accs {
		trie_acc := Account{nonce: acc.Nonce, balance: acc.Balance, code_size: uint64(len(acc.Code))}
		if trie_acc.code_size != 0 {
			code_hash := util.Hash(acc.Code)
			trie_acc.code_hash = code_hash
			self.db.db.PutCode(code_hash, acc.Code)
		}
		if len(acc.Storage) != 0 {
			var acc_tr_w trie.TrieWriter
			acc_tr_w.Init(AccountTrieIOPending{PendingBlockDB: &self.db, addr: &addr}, nil, trie.TrieWriterOpts{})
			for k, v := range acc.Storage {
				v := new(big.Int).SetBytes(v[:])
				assert.Holds(v.Sign() != 0)
				acc_tr_w.Put(util.Hash(k[:]), acc_trie_value(v))
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

func (self *StateTransitionService) TransitionState(tx_count int, params ...StateTransitionParams) (ret StateTransitionResult) {
	//ret.ExecutionResults = make([]vm.ExecutionResult, len(param.Transactions))
	self.db.blk_num = params[len(params)-1].Block.Number
	pending_accounts := make(map[common.Address]*pending_account, util.CeilPow2(tx_count*2))
	//pending_accounts := make(map[common.Address]*pending_account, util.CeilPow2(len(param.Transactions)*2))
	evm_state_sink := EVMStateOutput{
		OnAccountChanged: func(addr common.Address, change AccountChange) {
			acc := pending_accounts[addr]
			if acc == nil {
				acc = new(pending_account)
				pending_accounts[addr] = acc
				self.main_tr_w_executor.Do(func() {
					self.main_tr_w.Put(util.Hash(addr[:]), acc)
				})
			}
			acc.executor.Do(func() {
				acc.acc = change.Account
				if change.code_dirty {
					self.db.db.PutCode(change.code_hash, change.code)
				}
				if len(change.storage_dirty) == 0 {
					return
				}
				if acc.trie_w == nil {
					acc.trie_w = new(trie.TrieWriter).Init(
						AccountTrieIOPending{PendingBlockDB: &self.db, addr: &addr},
						acc.acc.storage_root_hash,
						trie.TrieWriterOpts{AnticipatedDepth: 18})
				}
				for k, v := range change.storage_dirty {
					if v.Sign() == 0 {
						acc.trie_w.Delete(util.Hash(k[:]))
					} else {
						acc.trie_w.Put(util.Hash(k[:]), acc_trie_value(v))
					}
				}
			})
		},
		OnAccountDeleted: func(addr common.Address) {
			pending_accounts[addr] = nil
			self.main_tr_w_executor.Do(func() {
				self.main_tr_w.Delete(util.Hash(addr[:]))
			})
		},
	}
	state := NewEVMState(&self.db, EvmStateOpts{
		AccountCacheSize:      len(pending_accounts) * 2,
		DirtyAccountCacheSize: len(pending_accounts),
	})
	for _, param := range params {
		rules := self.chain_cfg.Rules(param.Block.Number)
		evm_cfg := vm.NewEVMConfig(self.get_block_hash, param.Block, rules, self.opts_exec)
		if rules.IsDAOFork {
			misc.ApplyDAOHardFork(&state)
			state.Commit(rules.IsEIP158, evm_state_sink)
		}
		for i, cnt := TxIndex(0), TxIndex(len(param.Transactions)); i < cnt; i++ {
			if dbg.Debugging {
				fmt.Println("TX", i)
			}
			vm.Main(&evm_cfg, &state, &param.Transactions[i])
			//ret.ExecutionResults[i] = vm.Main(&evm_cfg, &state, &param.Transactions[i])
			state.Commit(rules.IsEIP158, evm_state_sink)
			//for addr, pending_acc := range pending_accounts {
			//	delete(pending_accounts, addr)
			//	if pending_acc == nil {
			//		continue
			//	}
			//	if !pending_acc.trie_w.IsZero() {
			//		pending_acc.acc.storage_root_hash = pending_acc.trie_w.Commit()
			//	}
			//	pending_acc.enc_storage, pending_acc.enc_hash = pending_acc.acc.EncodeForTrie()
			//}
			//fmt.Println(i, self.main_tr_w.Commit().Hex())
		}
		if !self.disable_block_rewards {
			ethash.AccumulateRewards(
				rules,
				ethash.BlockNumAndCoinbase{param.Block.Number, param.Block.Author},
				param.Uncles,
				state.AddBalance)
			state.Commit(rules.IsEIP158, evm_state_sink)
		}
	}
	for _, acc := range pending_accounts {
		if acc == nil {
			continue
		}
		acc := acc
		acc.executor.Do(func() {
			if acc.trie_w != nil {
				acc.acc.storage_root_hash = acc.trie_w.Commit()
			}
			acc.enc_storage, acc.enc_hash = acc.acc.EncodeForTrie()
		})
	}
	self.main_tr_w_executor.Do(func() {
		if h := self.main_tr_w.Commit(); h != nil {
			ret.StateRoot = *h
		} else {
			ret.StateRoot = empty_rlp_list_hash
		}
	})
	self.main_tr_w_executor.Synchronize()
	return
}

type pending_account struct {
	acc         Account
	trie_w      *trie.TrieWriter
	executor    util.SingleThreadExecutor
	enc_storage []byte
	enc_hash    []byte
}

func (self *pending_account) EncodeForTrie() (r0, r1 []byte) {
	self.executor.Synchronize()
	r0, r1 = self.enc_storage, self.enc_hash
	return
}
