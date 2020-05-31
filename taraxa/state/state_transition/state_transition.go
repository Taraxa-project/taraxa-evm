package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/consensus/ethash"
	"github.com/Taraxa-project/taraxa-evm/consensus/misc"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_concurrent_schedule"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/assert"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
	"math/big"
)

type StateTransition struct {
	db                            state_common.DB
	get_block_hash                vm.GetHashFunc
	chain_cfg                     state_common.ChainConfig
	main_tr_w                     trie.Writer
	main_tr_w_executor            util.SingleThreadExecutor
	acc_tr_writer_opts            trie.WriterCacheOpts
	pending_accounts              map[common.Address]*pending_account
	pending_accounts_keys         []common.Address
	evm_state                     state_evm.EVMState
	curr_blk_num                  types.BlockNum
	num_accounts_w_balance_change uint64
}

type CacheOpts struct {
	MainTrieWriterOpts        trie.WriterCacheOpts
	AccTrieWriterOpts         trie.WriterCacheOpts
	ExpectedMaxNumTrxPerBlock uint32
}

func (self *StateTransition) Init(
	db state_common.DB,
	get_block_hash vm.GetHashFunc,
	chain_cfg state_common.ChainConfig,
	curr_blk_num types.BlockNum,
	curr_state_root common.Hash,
	cache_opts CacheOpts,
) *StateTransition {
	self.db = db
	self.get_block_hash = get_block_hash
	self.chain_cfg = chain_cfg
	self.curr_blk_num = curr_blk_num
	curr_state_root_ptr := &curr_state_root
	if curr_state_root == state_common.EmptyRLPListHash || curr_state_root == common.ZeroHash {
		curr_state_root_ptr = nil
	}
	self.main_tr_w.Init(main_trie_db{StateTransition: self}, curr_state_root_ptr, cache_opts.MainTrieWriterOpts)
	self.acc_tr_writer_opts = cache_opts.AccTrieWriterOpts
	dirty_accs_per_block := uint32(util.CeilPow2(int(cache_opts.ExpectedMaxNumTrxPerBlock * 2)))
	accs_per_block := dirty_accs_per_block * 2
	self.pending_accounts = make(map[common.Address]*pending_account, accs_per_block)
	self.pending_accounts_keys = make([]common.Address, 0, accs_per_block)
	self.evm_state.Init(self, state_evm.CacheOpts{
		AccountsPrealloc:      accs_per_block,
		DirtyAccountsPrealloc: dirty_accs_per_block,
	})
	return self
}

type AccountMap = core.GenesisAlloc

func (self *StateTransition) ApplyAccounts(accs AccountMap) common.Hash {
	for addr, acc := range accs {
		trie_acc := state_common.Account{Nonce: acc.Nonce, Balance: acc.Balance, CodeSize: uint64(len(acc.Code))}
		if trie_acc.Balance == nil {
			trie_acc.Balance = common.Big0
		}
		if trie_acc.CodeSize != 0 {
			code_hash := keccak256.Hash(acc.Code)
			trie_acc.CodeHash = code_hash
			self.db.PutCode(code_hash, acc.Code)
		}
		if len(acc.Storage) != 0 {
			var acc_tr_w trie.Writer
			addr := addr
			acc_tr_w.Init(account_trie_db{StateTransition: self, addr: &addr}, nil, self.acc_tr_writer_opts)
			for k, v := range acc.Storage {
				v := new(big.Int).SetBytes(v[:])
				assert.Holds(v.Sign() != 0)
				acc_tr_w.Put(keccak256.Hash(k[:]), state_common.EncodeAccountTrieValue(v))
			}
			trie_acc.StorageRootHash = acc_tr_w.Commit()
		}
		self.main_tr_w.Put(keccak256.Hash(addr[:]), state_common.AccountEncoder{&trie_acc})
	}
	if ret := self.main_tr_w.Commit(); ret != nil {
		return *ret
	}
	return state_common.EmptyRLPListHash
}

type UncleBlock = ethash.BlockNumAndCoinbase
type AddressAndBalance = struct {
	Addr    common.Address
	Balance *big.Int
}
type Result struct {
	StateRoot        common.Hash
	ExecutionResults []vm.ExecutionResult
	BalanceChanges   []AddressAndBalance
}

func (self *StateTransition) ApplyBlock(
	evm_block *vm.BlockWithoutNumber,
	transactions []vm.Transaction,
	uncles []UncleBlock,
	concurrent_schedule state_concurrent_schedule.ConcurrentSchedule,
) (ret Result) {
	ret.ExecutionResults = make([]vm.ExecutionResult, len(transactions))
	self.curr_blk_num++
	rules := self.chain_cfg.ETHChainConfig.Rules(self.curr_blk_num)
	if rules.IsDAOFork {
		misc.ApplyDAOHardFork(&self.evm_state)
		self.evm_state.Commit(rules.IsEIP158, self)
	}
	evm_blk := vm.Block{self.curr_blk_num, *evm_block}
	evm_cfg := vm.NewEVMConfig(self.get_block_hash, &evm_blk, rules, self.chain_cfg.ExecutionOptions)
	for i, cnt := state_common.TxIndex(0), state_common.TxIndex(len(transactions)); i < cnt; i++ {
		ret.ExecutionResults[i] = vm.Main(&evm_cfg, &self.evm_state, &transactions[i])
		self.evm_state.Commit(rules.IsEIP158, self)
	}
	if !self.chain_cfg.DisableBlockRewards {
		ethash.AccumulateRewards(
			rules,
			ethash.BlockNumAndCoinbase{self.curr_blk_num, evm_block.Author},
			uncles,
			self.evm_state.AddBalance)
		self.evm_state.Commit(rules.IsEIP158, self)
	}
	ret.BalanceChanges = make([]AddressAndBalance, self.num_accounts_w_balance_change)
	balance_changes_pos := 0
	for _, addr := range self.pending_accounts_keys {
		acc := self.pending_accounts[addr]
		if acc == nil {
			continue
		}
		delete(self.pending_accounts, addr)
		balance_changes_pos_ := balance_changes_pos
		if acc.balance_dirty {
			ret.BalanceChanges[balance_changes_pos_].Addr = addr
			balance_changes_pos++
		}
		acc.executor.Do(func() {
			if acc.trie_w != nil {
				acc.acc.StorageRootHash = acc.trie_w.Commit()
			}
			acc.enc_storage, acc.enc_hash = state_common.AccountEncoder{&acc.acc}.EncodeForTrie()
			if acc.balance_dirty {
				ret.BalanceChanges[balance_changes_pos_].Balance = acc.acc.Balance
			}
		})
	}
	self.main_tr_w_executor.Do(func() {
		if h := self.main_tr_w.Commit(); h != nil {
			ret.StateRoot = *h
		} else {
			ret.StateRoot = state_common.EmptyRLPListHash
		}
	})
	self.evm_state.Reset()
	self.pending_accounts_keys = self.pending_accounts_keys[:0]
	self.num_accounts_w_balance_change = 0
	self.main_tr_w_executor.Synchronize()
	return
}

func (self *StateTransition) GetCode(hash *common.Hash) []byte {
	return self.db.GetCode(hash)
}

func (self *StateTransition) GetAccount(addr *common.Address) (ret state_common.Account, present bool) {
	enc_storage := self.db.GetMainTrieValueLatest(keccak256.Hash(addr[:]))
	if present = len(enc_storage) != 0; present {
		state_common.DecodeAccount(&ret, enc_storage)
	}
	return
}

func (self *StateTransition) GetAccountStorage(addr *common.Address, key *common.Hash) *big.Int {
	if enc_storage := self.db.GetAccountTrieValueLatest(addr, keccak256.Hash(key[:])); len(enc_storage) != 0 {
		_, val, _ := rlp.MustSplit(enc_storage)
		return new(big.Int).SetBytes(val)
	}
	return common.Big0
}

func (self *StateTransition) OnAccountChanged(addr common.Address, change state_evm.AccountChange) {
	acc := self.pending_accounts[addr]
	if acc == nil {
		self.pending_accounts_keys = append(self.pending_accounts_keys, addr)
		acc = new(pending_account)
		self.pending_accounts[addr] = acc
		self.main_tr_w_executor.Do(func() {
			self.main_tr_w.Put(keccak256.Hash(addr[:]), acc)
		})
	}
	if change.BalanceDirty && !acc.balance_dirty {
		acc.balance_dirty = true
		self.num_accounts_w_balance_change++
	}
	acc.executor.Do(func() {
		acc.acc = change.Account
		if change.CodeDirty {
			self.db.PutCode(change.CodeHash, change.Code)
		}
		if len(change.StorageDirty) == 0 {
			return
		}
		if acc.trie_w == nil {
			acc.trie_w = new(trie.Writer).Init(
				account_trie_db{StateTransition: self, addr: &addr},
				acc.acc.StorageRootHash,
				self.acc_tr_writer_opts)
		}
		for k, v := range change.StorageDirty {
			if v.Sign() == 0 {
				acc.trie_w.Delete(keccak256.Hash(k[:]))
			} else {
				acc.trie_w.Put(keccak256.Hash(k[:]), state_common.EncodeAccountTrieValue(v))
			}
		}
	})
}

func (self *StateTransition) OnAccountDeleted(addr common.Address) {
	delete(self.pending_accounts, addr)
	self.main_tr_w_executor.Do(func() {
		self.main_tr_w.Delete(keccak256.Hash(addr[:]))
	})
}
