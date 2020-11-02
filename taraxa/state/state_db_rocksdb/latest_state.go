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
	batch           *gorocksdb.WriteBatch
	writer_thread   goroutines.SingleThreadExecutor
	committed_blk_n types.BlockNum
	pending_blk_n   types.BlockNum
	blk_num_mu      sync.Mutex
}

func (self *latest_state) Init(db *DB) *latest_state {
	self.DB = db
	self.batch = gorocksdb.NewWriteBatch()
	self.writer_thread.Init(1024) // 8KB
	self.committed_blk_n = self.GetCommittedDescriptor().BlockNum
	self.pending_blk_n = self.committed_blk_n
	return self
}

func (self *latest_state) Close() {
	self.writer_thread.JoinAndClose()
	self.batch.Destroy()
}

func (self *latest_state) GetCommittedDescriptor() (ret state_db.StateDescriptor) {
	v_slice, err := self.db.Get(self.opts_r, last_committed_desc_key)
	util.PanicIfNotNil(err)
	defer v_slice.Free()
	ret.BlockNum = types.BlockNumberNIL
	if v := v_slice.Data(); len(v) != 0 {
		rlp.MustDecodeBytes(v, &ret)
	}
	return
}

func (self *latest_state) BeginPendingBlock() state_db.PendingBlockState {
	defer util.LockUnlock(&self.blk_num_mu)()
	self.pending_blk_n++
	var keybuf TrieValueKey
	keybuf.SetBlockNum(self.pending_blk_n)
	return &pending_block_state{block_state_reader{self.DB, self.committed_blk_n}, self.pending_blk_n, keybuf}
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
	self.blk_num_mu.Lock()
	committed_blk_n := self.pending_blk_n
	self.committed_blk_n = committed_blk_n
	self.blk_num_mu.Unlock()
	self.writer_thread.Submit(func() {
		self.batch.Put(last_committed_desc_key, rlp.MustEncodeToBytes(state_db.StateDescriptor{
			BlockNum:  committed_blk_n,
			StateRoot: state_root,
		}))
		err = self.db.Write(self.opts_w, self.batch)
		self.batch.Clear()
		self.reset_itr_pools()
	})
	self.writer_thread.Join() // TODO completely async
	return
}
