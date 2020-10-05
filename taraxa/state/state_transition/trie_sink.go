package state_transition

import (
	"fmt"
	"runtime"

	"github.com/Taraxa-project/taraxa-evm/dbg"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_common"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigconv"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_evm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/goroutines"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type TrieSink struct {
	thread_main_trie_write     goroutines.SingleThreadExecutor
	threads_account_trie_write goroutines.SequentialTaskGroupExecutor
	io                         state_db.ReadWriter
	main_trie_writer           trie.Writer
}
type TrieSinkOpts struct {
	MainTrie trie.WriterOpts
}

func (self *TrieSink) Init(state_root *common.Hash, opts TrieSinkOpts) *TrieSink {
	if state_common.IsEmptyStateRoot(state_root) {
		state_root = nil
	}
	self.main_trie_writer.Init(state_db.MainTrieSchema{}, state_root, opts.MainTrie)
	self.thread_main_trie_write.Init(1024)                       // 8KB
	self.threads_account_trie_write.Init(1024, runtime.NumCPU()) // 8KB
	return self
}

func (self *TrieSink) Close() {
	self.thread_main_trie_write.JoinAndClose()
	self.threads_account_trie_write.Close()
}

func (self *TrieSink) SetIO(io state_db.ReadWriter) {
	self.io = io
}

func (self *TrieSink) StartMutation(addr *common.Address) state_evm.AccountMutation {
	return &TrieSinkAccountMutation{
		host:   self,
		addr:   addr,
		thread: self.threads_account_trie_write.NewGroup(8), // negligible space
	}
}

func (self *TrieSink) Delete(addr *common.Address) {
	if dbg.Debug {
		fmt.Println("del", addr.Hex())
	}
	io := self.io
	self.thread_main_trie_write.Submit(func() {
		self.main_trie_writer.Delete(state_db.MainTrieIOAdapter{io}, keccak256.Hash(addr[:]))
	})
}

type TrieSinkAccountMutation struct {
	host        *TrieSink
	addr        *common.Address
	thread      goroutines.SequentialTaskGroup
	acc         state_db.Account
	pending     bool
	enc_storage []byte
	enc_hash    []byte
	trie_writer *trie.Writer
}

func (self *TrieSinkAccountMutation) Update(upd state_evm.AccountChange) {
	if dbg.Debug {
		fmt.Println("upd", self.addr.Hex(), dbg.JSON(upd))
	}
	io := self.host.io
	if upd.CodeDirty {
		io.Put(state_db.COL_code, upd.CodeHash, upd.Code)
	}
	if !self.pending {
		self.pending = true
		self.host.thread_main_trie_write.Submit(func() {
			self.host.main_trie_writer.Put(state_db.MainTrieIOAdapter{io}, keccak256.Hash(self.addr[:]), self)
		})
	}
	self.thread.Submit(func() {
		self.acc = upd.Account
		if len(upd.StorageDirty) == 0 && len(upd.RawStorageDirty) == 0 {
			return
		}
		if self.trie_writer == nil {
			self.trie_writer = new(trie.Writer).
				Init(state_db.AccountTrieSchema{}, upd.StorageRootHash, trie.WriterOpts{})
		}
		var big_conv bigconv.BigConv
		trie_io := state_db.AccountTrieIOAdapter{self.addr, io}
		for k, v := range upd.StorageDirty {
			if k_h := keccak256.Hash(big_conv.ToHash(k.Int())[:]); v.Sign() == 0 {
				self.trie_writer.Delete(trie_io, k_h)
			} else {
				self.trie_writer.Put(trie_io, k_h, state_db.NewAccStorageTrieValue(v.Bytes()))
			}
		}
		for k, v := range upd.RawStorageDirty {
			if k_h := keccak256.Hash(k[:]); len(v) == 0 {
				self.trie_writer.Delete(trie_io, k_h)
			} else {
				self.trie_writer.Put(trie_io, k_h, state_db.NewAccStorageTrieValue(v))
			}
		}
	})
}

func (self *TrieSinkAccountMutation) Commit() {
	if !self.pending {
		return
	}
	self.pending = false
	io := self.host.io
	self.thread.Submit(func() {
		if self.trie_writer != nil {
			self.acc.StorageRootHash = self.trie_writer.Commit(state_db.AccountTrieIOAdapter{self.addr, io})
		}
		self.enc_storage, self.enc_hash = self.acc.EncodeForTrie()
	})
}

func (self *TrieSinkAccountMutation) EncodeForTrie() (r0, r1 []byte) {
	self.thread.Join()
	return self.enc_storage, self.enc_hash
}

func (self *TrieSink) Commit() (state_root common.Hash) {
	state_root = state_common.EmptyRLPListHash
	io := self.io
	self.thread_main_trie_write.Submit(func() {
		if state_root_p := self.main_trie_writer.Commit(state_db.MainTrieIOAdapter{io}); state_root_p != nil {
			state_root = *state_root_p
		}
	})
	self.thread_main_trie_write.Join()
	return
}
