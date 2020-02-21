package eth_trie

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"github.com/Taraxa-project/taraxa-evm/trie"
	"github.com/emicklei/dot"
	"github.com/tecbot/gorocksdb"
)

// TODO eliminate caching
type TrieState struct {
	ethdb_adapter
	block_count  state.BlockOrdinal
	current_root common.Hash
	Dot_g        *dot.Graph
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
	trie, err_0 := self.getTrie(block_ordinal)
	if err = err_0; err != nil {
		return
	}
	self.batch = gorocksdb.NewWriteBatch()
	defer func() {
		self.batch = nil
	}()
	for _, e := range state_change {
		if err = trie.Insert(e.K, e.V); err != nil {
			return
		}
	}
	if self.current_root, err = trie.Commit(); err != nil {
		return
	}
	checksum = self.current_root.Bytes()
	block_num_bytes := util.ENC_b_endian_compact_64(self.block_count)
	self.db.BatchPutCol(self.batch, COL_default, last_block_num_key, block_num_bytes)
	self.db.BatchPutCol(self.batch, COL_block_num_to_root, block_num_bytes, checksum)
	if err = trie.GetDB().Commit(); err != nil {
		return
	}
	if err = self.db.Commit(self.batch); err != nil {
		return
	}
	block_ordinal = self.block_count
	self.block_count++
	return
}

func (self *TrieState) getTrie(block_ordinal state.BlockOrdinal) (ret *trie.Trie, err error) {
	defer func() {
		if ret != nil {
			ret.Dot_g = self.Dot_g
		}
	}()
	trie_db := trie.NewDatabase(&self.ethdb_adapter)
	if self.block_count == 0 || block_ordinal == self.block_count-1 {
		return trie.New(self.current_root, trie_db)
	}
	var root []byte
	root, err = self.db.GetCol(COL_block_num_to_root, util.ENC_b_endian_compact_64(block_ordinal))
	if err != nil {
		return
	}
	return trie.New(common.BytesToHash(root), trie_db)
}

func (self *TrieState) Get(block_ordinal state.BlockOrdinal, k []byte) ([]byte, error) {
	t, err := self.getTrie(block_ordinal)
	if err != nil {
		return nil, err
	}
	return t.Get(k)
}

func (self *TrieState) GetWithProof(block_ordinal state.BlockOrdinal, k []byte) (state.ValueProof, error) {
	t, err := self.getTrie(block_ordinal)
	if err != nil {
		return nil, err
	}
	proof_db := ethdb.NewMemDatabase()
	return &Proof{proof_db}, t.Prove(k, 0, proof_db)
}

func (self *TrieState) Close() {
	self.db.Close()
}

var last_block_num_key = []byte("last_block")
