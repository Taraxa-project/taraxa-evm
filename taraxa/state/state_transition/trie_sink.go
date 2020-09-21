package state_transition

import (
	"runtime"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigconv"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/goroutines"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type TrieSink struct {
	thread                     goroutines.SingleThreadExecutor
	thread_main_trie_write     goroutines.SingleThreadExecutor
	threads_account_trie_write goroutines.SequentialTaskGroupExecutor
	acc_tr_writer_opts         trie.WriterCacheOpts
	main_trie_db               state_trie.MainTrieDB
	main_trie_writer           trie.Writer
}

type TrieWriterOpts struct {
	MainTrieWriterOpts trie.WriterCacheOpts
	AccTrieWriterOpts  trie.WriterCacheOpts
}
type TrieSinkOpts struct {
	TrieWriters              TrieWriterOpts
	NumDirtyAccountsToBuffer uint32
}

func (self *TrieSink) Init(state_root *common.Hash, opts TrieSinkOpts) *TrieSink {
	if state_root != nil && (*state_root == state_common.EmptyRLPListHash || *state_root == common.ZeroHash) {
		state_root = nil
	}
	self.main_trie_writer.Init(&self.main_trie_db, state_root, opts.TrieWriters.MainTrieWriterOpts)
	self.acc_tr_writer_opts = opts.TrieWriters.AccTrieWriterOpts
	self.thread.Init(64)
	self.thread_main_trie_write.Init(opts.NumDirtyAccountsToBuffer)
	self.threads_account_trie_write.Init(opts.NumDirtyAccountsToBuffer, runtime.NumCPU())
	return self
}

func (self *TrieSink) BeginBatch(db_tx state_common.BlockCreationTransaction) {
	self.thread.Submit(func() {
		// TODO this is just for the test
		self.thread_main_trie_write.Join()
		self.main_trie_db.SetTransaction(db_tx)
	})
}

func (self *TrieSink) StartMutation(addr *common.Address) state_evm.AccountMutation {
	return &TrieSinkAccountMutation{host: self, addr: addr, thread: self.threads_account_trie_write.NewGroup(4)}
}

func (self *TrieSink) Delete(addr *common.Address) {
	self.thread.Submit(func() {
		self.thread_main_trie_write.Submit(func() {
			self.main_trie_writer.Delete(keccak256.Hash(addr[:]))
		})
	})
}

type TrieSinkAccountMutation struct {
	host        *TrieSink
	addr        *common.Address
	thread      goroutines.SequentialTaskGroup
	acc         state_trie.Account
	pending     bool
	enc_storage []byte
	enc_hash    []byte
	*TrieSinkAccountMutationTrieState
}
type TrieSinkAccountMutationTrieState struct {
	trie_db     state_trie.AccountTrieDB
	trie_writer trie.Writer
}

func (self *TrieSinkAccountMutation) Update(upd state_evm.AccountChange) {
	self.host.thread.Submit(func() {
		db_tx := self.host.main_trie_db.GetTransaction()
		if upd.CodeDirty {
			db_tx.PutCode(upd.CodeHash, upd.Code.Value())
		}
		if !self.pending {
			self.pending = true
			self.host.thread_main_trie_write.Submit(func() {
				self.host.main_trie_writer.Put(keccak256.Hash(self.addr[:]), self)
			})
		}
		self.thread.Submit(func() {
			self.acc = upd.Account
			if len(upd.StorageDirty) == 0 && len(upd.RawStorageDirty) == 0 {
				return
			}
			if self.TrieSinkAccountMutationTrieState == nil {
				self.TrieSinkAccountMutationTrieState = new(TrieSinkAccountMutationTrieState)
				self.trie_writer.Init(
					self.trie_db.Init(self.addr),
					upd.StorageRootHash,
					self.host.acc_tr_writer_opts)
			}
			// TODO don't do on every call
			self.trie_db.SetTransaction(db_tx)
			var big_conv bigconv.BigConv
			for k, v := range upd.StorageDirty {
				if k_h := keccak256.Hash(big_conv.ToHash(k.Int())[:]); v.Sign() == 0 {
					self.trie_writer.Delete(k_h)
				} else {
					self.trie_writer.Put(k_h, state_trie.NewAccStorageTrieValue(v.Bytes()))
				}
			}
			for k, v := range upd.RawStorageDirty {
				if k_h := keccak256.Hash(k[:]); len(v) == 0 {
					self.trie_writer.Delete(k_h)
				} else {
					self.trie_writer.Put(k_h, state_trie.NewAccStorageTrieValue(v))
				}
			}
		})
	})
}

func (self *TrieSinkAccountMutation) Commit() {
	if !self.pending {
		return
	}
	self.pending = false
	self.thread.Submit(func() {
		if self.TrieSinkAccountMutationTrieState != nil {
			self.acc.StorageRootHash = self.trie_writer.Commit()
		}
		self.enc_storage, self.enc_hash = self.acc.EncodeForTrie()
	})
}

func (self *TrieSinkAccountMutation) EncodeForTrie() (r0, r1 []byte) {
	self.thread.Join()
	return self.enc_storage, self.enc_hash
}

func (self *TrieSink) CommitSync(batch state_evm.AccountMutations) (state_root *common.Hash) {
	self.thread.Submit(func() {
		batch.ForEachMutationWithDuplicates(func(writer state_evm.AccountMutation) {
			writer.(*TrieSinkAccountMutation).Commit()
		})
		self.thread_main_trie_write.Submit(func() {
			if state_root = self.main_trie_writer.Commit(); state_root == nil {
				state_root = &state_common.EmptyRLPListHash
			}
		})
		self.thread_main_trie_write.Join()
	})
	self.thread.Join()
	return
}
