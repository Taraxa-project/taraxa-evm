package state_db_rocksdb

import (
	"sync"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/goroutines"
	"github.com/tecbot/gorocksdb"
)

var last_committed_desc_key = []byte("last_committed_descriptor")

type latest_state struct {
	*DB
	batch         *gorocksdb.WriteBatch
	writer_thread goroutines.SingleThreadExecutor
	state_desc    state_db.StateDescriptor
	pending_blk_n types.BlockNum
	state_desc_mu sync.RWMutex
}

func (self *latest_state) Init(db *DB) *latest_state {
	self.DB = db
	self.batch = gorocksdb.NewWriteBatch()
	self.writer_thread.Init(1024) // 8KB
	state_desc_raw, err := self.db.Get(self.opts_r, last_committed_desc_key)
	util.PanicIfNotNil(err)
	defer state_desc_raw.Free()
	self.state_desc.BlockNum = types.BlockNumberNIL
	if v := state_desc_raw.Data(); len(v) != 0 {
		rlp.MustDecodeBytes(v, &self.state_desc)
	}
	self.pending_blk_n = self.state_desc.BlockNum
	return self
}

func (self *latest_state) Close() {
	self.writer_thread.JoinAndClose()
	self.batch.Destroy()
}

func (self *latest_state) GetCommittedDescriptor() (ret state_db.StateDescriptor) {
	defer util.LockUnlock(self.state_desc_mu.RLocker())()
	return self.state_desc
}

func (self *latest_state) BeginPendingBlock() state_db.PendingBlockState {
	defer util.LockUnlock(&self.state_desc_mu)()
	self.pending_blk_n++
	var keybuf TrieValueKey
	keybuf.SetBlockNum(self.pending_blk_n)
	return &pending_block_state{block_state_reader{self.DB, self.state_desc.BlockNum}, self.pending_blk_n, keybuf}
}

type pending_block_state struct {
	block_state_reader
	blk_n              types.BlockNum
	trie_value_key_buf TrieValueKey
}

func (self *pending_block_state) Put(col state_db.Column, k *common.Hash, v []byte) {
	self.latest_state.writer_thread.Submit(func() {
		if col == state_db.COL_acc_trie_value || col == state_db.COL_main_trie_value {
			self.trie_value_key_buf.SetKey(k)
			self.latest_state.batch.PutCF(self.cf_handles[col], self.trie_value_key_buf[:], v)
		} else {
			self.latest_state.batch.PutCF(self.cf_handles[col], k[:], v)
		}
	})
}

func (self *pending_block_state) GetNumber() types.BlockNum {
	return self.blk_n
}

func (self *latest_state) Commit(state_root common.Hash) (err error) {
	state_desc := &state_db.StateDescriptor{BlockNum: self.pending_blk_n, StateRoot: state_root}
	self.writer_thread.Submit(func() {
		self.batch.Put(last_committed_desc_key, rlp.MustEncodeToBytes(state_desc))
		if err = self.db.Write(self.opts_w, self.batch); err == nil {
			self.state_desc_mu.Lock()
			self.state_desc = *state_desc
			self.state_desc_mu.Unlock()
			self.reset_itr_pools()
		}
		self.batch.Clear()
	})
	self.writer_thread.Join() // TODO completely async
	return
}
