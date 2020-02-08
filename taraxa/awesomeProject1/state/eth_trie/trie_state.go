package eth_trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"github.com/tecbot/gorocksdb"
)

// TODO eliminate caching
type TrieState struct {
	db            *rocksdb_ext.RocksDBExt
	block_count   state.BlockOrdinal
	ethdb_adapter ethdb_adapter
	trie_db       *trie.Database
	current_root  common.Hash
}
type TrieStateConfig = struct {
	rocksdb_ext.RocksDBExtDBConfig
	ColumnOpts [COL_count]rocksdb_ext.RocksDBExtColumnOpts
}

const (
	COL_default = iota
	COL_block_num_to_root
	COL_state_entries
	COL_count
)

func NewTrieState(cfg *TrieStateConfig) (self *TrieState, err error) {
	self = new(TrieState)
	self.db, err = rocksdb_ext.NewRocksDBExt(&rocksdb_ext.RocksDBExtConfig{
		cfg.RocksDBExtDBConfig,
		cfg.ColumnOpts[:],
	})
	if err != nil {
		return
	}
	self.ethdb_adapter.col = COL_state_entries
	self.ethdb_adapter.db = self.db
	self.trie_db = trie.NewDatabase(&self.ethdb_adapter)
	block_num_bytes, err_0 := self.db.GetCol(COL_default, last_block_num_key)
	if err = err_0; err_0 != nil {
		return
	}
	if block_num_bytes != nil {
		self.block_count = util.DEC_b_endian_compact_64(block_num_bytes)
		root_bytes, err_1 := self.db.GetCol(COL_block_num_to_root, block_num_bytes)
		if err = err_1; err_1 != nil {
			return
		}
		self.current_root = common.BytesToHash(root_bytes)
	}
	return
}

func (self *TrieState) CommitBlock(state_change state.StateChange) (block_ordinal state.BlockOrdinal, checksum []byte, err error) {
	trie, err_0 := trie.New(self.current_root, self.trie_db)
	if err = err_0; err != nil {
		return
	}
	batch := gorocksdb.NewWriteBatch()
	self.ethdb_adapter.batch = batch
	for _, e := range state_change {
		trie.TryUpdate(e.K, e.V)
	}
	if self.current_root, err = trie.Commit(); err != nil {
		return
	}
	defer func() {
		self.ethdb_adapter.batch = nil
	}()
	if err = self.trie_db.Commit(); err != nil {
		return
	}
	checksum = self.current_root.Bytes()
	block_num_bytes := util.ENC_b_endian_compact_64(self.block_count)
	self.db.BatchPutCol(batch, COL_default, last_block_num_key, block_num_bytes)
	self.db.BatchPutCol(batch, COL_block_num_to_root, block_num_bytes, checksum)
	if err = self.db.Commit(batch); err != nil {
		return
	}
	block_ordinal = self.block_count
	self.block_count++
	return
}

func (self *TrieState) getTrie(block_ordinal state.BlockOrdinal) (*trie.Trie, error) {
	if block_ordinal == self.block_count-1 {
		return  trie.New(self.current_root, self.trie_db)
	}
	root, err := self.db.GetCol(COL_block_num_to_root, util.ENC_b_endian_compact_64(block_ordinal))
	if err != nil {
		return nil, err
	}
	return trie.New(common.BytesToHash(root), self.trie_db)
}

func (self *TrieState) Get(block_ordinal state.BlockOrdinal, k []byte) ([]byte, error) {
	t, err := self.getTrie(block_ordinal)
	if err != nil {
		return nil, err
	}
	return t.TryGet(k)
}

func (self *TrieState) GetWithProof(block_ordinal state.BlockOrdinal, k []byte) (state.ValueProof, error) {
	t, err := self.getTrie(block_ordinal)
	if err != nil {
		return nil, err
	}
	proof_db := ethdb.NewMemDatabase()
	return &Proof{proof_db}, t.Prove(k, 0, proof_db)
}

var last_block_num_key = []byte("last_block")
