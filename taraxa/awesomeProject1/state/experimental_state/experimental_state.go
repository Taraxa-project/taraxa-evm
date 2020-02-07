package experimental_state

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"github.com/tecbot/gorocksdb"
	"math"
	"unsafe"
)

type ExperimentalState struct {
	db                   *rocksdb_ext.RocksDBExt
	block_count          state.BlockOrdinal
	entry_count          EntryOrdinal
	digest               []byte
	absent_node_hash_gen AbsentNodeHashGenerator
}
type ExperimentalStateConfig struct {
	rocksdb_ext.RocksDBExtDBConfig
	ColumnOpts           [COL_count]rocksdb_ext.RocksDBExtColumnOpts
	absent_node_hash_gen AbsentNodeHashGenerator
}
type MerkleLevelOrdinal = byte
type EntryOrdinal = uint64
type MerkleLevel = map[EntryOrdinal][]byte
type AbsentNodeHashGenerator = func(MerkleLevelOrdinal, EntryOrdinal, state.BlockOrdinal) []byte

const Arity = 2
const BlockOrdinalSize = unsafe.Sizeof(state.BlockOrdinal(0))
const EntryOrdinalSize = unsafe.Sizeof(EntryOrdinal(0))
const MerkleLevelOrdinalSize = unsafe.Sizeof(MerkleLevelOrdinal(0))
const MerkleKeyPrefixSize = MerkleLevelOrdinalSize + EntryOrdinalSize
const MerkleKeySize = MerkleLevelOrdinalSize + EntryOrdinalSize + BlockOrdinalSize

const (
	COL_default = iota
	COL_merkle
	COL_entries
	COL_block_entry_count
	COL_count
)

func NewExperimentalState(cfg *ExperimentalStateConfig) (self *ExperimentalState, err error) {
	self = new(ExperimentalState)
	self.db, err = rocksdb_ext.NewRocksDBExt(&rocksdb_ext.RocksDBExtConfig{
		cfg.RocksDBExtDBConfig,
		cfg.ColumnOpts[:],
	})
	if err != nil {
		return
	}
	self.digest = keccak256_0
	self.absent_node_hash_gen = cfg.absent_node_hash_gen
	if self.absent_node_hash_gen == nil {
		self.absent_node_hash_gen = func(MerkleLevelOrdinal, EntryOrdinal, state.BlockOrdinal) []byte {
			return keccak256_0
		}
	}
	block_ordinal_enc_compact, err_0 := self.db.GetCol(COL_default, last_block_ordinal_key)
	if err = err_0; err_0 != nil || block_ordinal_enc_compact == nil {
		return
	}
	self.block_count = util.DEC_b_endian_compact_64(block_ordinal_enc_compact) + 1
	entry_count_enc_compact, err_1 := self.db.GetCol(COL_block_entry_count, block_ordinal_enc_compact)
	if err = err_1; err_1 != nil {
		return
	}
	self.entry_count = util.DEC_b_endian_compact_64(entry_count_enc_compact)
	merkle_root_lvl_ordinal := merkle_root_lvl_ordinal(self.entry_count)
	self.digest, err = self.db.GetCol(COL_merkle, merkle_key(merkle_root_lvl_ordinal, b_endian_64_0, util.ENC_b_endian_64(self.block_count-1)))
	return
}

// TODO state_change order deterministically defined by the caller layers, possibly VRF
// TODO defragmentation sortition could be deterministically defined by the caller, possibly VRF
// TODO empty (new subtree after the doubling) merkle node hashes deterministically defined by the caller
// TODO use hashed key, possibly store preimages

// TODO reuse buffers more
// TODO parallelism
func (self *ExperimentalState) CommitBlock(state_change state.StateChange) (block_ordinal state.BlockOrdinal, digest []byte, err error) {
	block_ordinal = self.block_count
	digest = self.digest
	block_ordinal_enc_compact := util.ENC_b_endian_compact_64(block_ordinal)
	if len(state_change) == 0 {
		if err = self.db.PutCol(COL_default, last_block_ordinal_key, block_ordinal_enc_compact); err == nil {
			self.block_count++
		}
		return
	}
	entry_count := self.entry_count
	block_ordinal_b_endian := util.ENC_b_endian_64(block_ordinal)
	merkle_lvl_ordinal := MerkleLevelOrdinal(0)
	merkle_lvl := make(MerkleLevel, len(state_change))
	batch := gorocksdb.NewWriteBatch()
	for _, entry := range state_change {
		entry_k := append(entry.K, block_ordinal_b_endian...)
		found_k, found_v, err_0 := self.db.Find(COL_entries, entry_k, true)
		if err = err_0; err_0 != nil {
			return
		}
		var entry_ordinal EntryOrdinal
		var entry_ordinal_b_endian []byte
		if bytes.HasPrefix(found_k, entry.K) {
			entry_ordinal_b_endian = found_v[:EntryOrdinalSize]
			entry_ordinal = binary.BigEndian.Uint64(entry_ordinal_b_endian)
		} else {
			entry_ordinal = entry_count
			entry_ordinal_b_endian = util.ENC_b_endian_64(entry_ordinal)
			entry_count++
		}
		self.db.BatchPutCol(batch, COL_entries, entry_k, append(entry_ordinal_b_endian, entry.V...))
		merkle_db_k := merkle_key(merkle_lvl_ordinal, entry_ordinal_b_endian, block_ordinal_b_endian)
		entry_hash := crypto.Keccak256(entry.K, entry_ordinal_b_endian, entry.V)
		self.db.BatchPutCol(batch, COL_merkle, merkle_db_k, entry_hash)
		merkle_lvl[entry_ordinal] = entry_hash
	}
	sibling_hashes_buf := make([][]byte, Arity)
	merkle_root_lvl_ordinal := merkle_root_lvl_ordinal(entry_count)
	for ; merkle_lvl_ordinal < merkle_root_lvl_ordinal; merkle_lvl_ordinal++ {
		// TODO precise allocation
		next_lvl := make(MerkleLevel)
		next_lvl_ordinal := merkle_lvl_ordinal + 1
		for entry_ordinal, hash := range merkle_lvl {
			delete(merkle_lvl, entry_ordinal)
			entry_local_pos := entry_ordinal % Arity
			sibling_local_pos := 1 - entry_local_pos
			sibling_ordinal := entry_ordinal - entry_local_pos + sibling_local_pos
			sibling_hash, ok := merkle_lvl[sibling_ordinal]
			if ok {
				delete(merkle_lvl, sibling_ordinal)
			} else {
				sibling_db_key := merkle_key(merkle_lvl_ordinal, util.ENC_b_endian_64(sibling_ordinal), block_ordinal_b_endian)
				found_key, found_val, err_1 := self.db.Find(COL_merkle, sibling_db_key, true)
				if err = err_1; err_1 != nil {
					return
				}
				if bytes.HasPrefix(found_key, sibling_db_key[:MerkleKeyPrefixSize]) && found_val != nil {
					sibling_hash = found_val
				} else {
					sibling_hash = self.absent_node_hash_gen(merkle_lvl_ordinal, sibling_ordinal, block_ordinal)
					self.db.BatchPutCol(batch, COL_merkle, sibling_db_key, sibling_hash)
				}
			}
			sibling_hashes_buf[entry_local_pos] = hash
			sibling_hashes_buf[sibling_local_pos] = sibling_hash
			parent_hash := crypto.Keccak256(sibling_hashes_buf...)
			parent_ordnal := entry_ordinal / Arity
			parent_db_key := merkle_key(next_lvl_ordinal, util.ENC_b_endian_64(parent_ordnal), block_ordinal_b_endian)
			self.db.BatchPutCol(batch, COL_merkle, parent_db_key, parent_hash)
			next_lvl[parent_ordnal] = parent_hash
		}
		merkle_lvl = next_lvl
	}
	digest = merkle_lvl[0]
	self.db.BatchPutCol(batch, COL_block_entry_count, block_ordinal_enc_compact, util.ENC_b_endian_compact_64(entry_count))
	self.db.BatchPutCol(batch, COL_default, last_block_ordinal_key, block_ordinal_enc_compact)
	if err = self.db.Write(rocksdb_ext.Default_opts_w, batch); err != nil {
		return
	}
	self.entry_count = entry_count
	self.digest = digest
	self.block_count++
	return
}

func (self *ExperimentalState) Get(block_ordinal state.BlockOrdinal, k []byte) (ret []byte, err error) {
	if self.block_count <= block_ordinal {
		return
	}
	db_k := append(k, util.ENC_b_endian_64(block_ordinal)...)
	found_key, found_val, err_0 := self.db.Find(COL_entries, db_k, true)
	if err = err_0; err_0 != nil || !bytes.HasPrefix(found_key, k) || found_val == nil {
		return
	}
	return found_val[EntryOrdinalSize:], nil
}

func (self *ExperimentalState) GetWithProof(block_ordinal state.BlockOrdinal, k []byte) (proof state.ValueProof, err error) {
	if self.block_count <= block_ordinal {
		return
	}
	found_key, found_v, err_0 := self.db.Find(COL_entries, append(k, util.ENC_b_endian_64(block_ordinal)...), true)
	if err = err_0; err_0 != nil || !bytes.HasPrefix(found_key, k) || found_v == nil {
		return
	}
	entry_count := self.entry_count
	if self.block_count-1 != block_ordinal {
		entry_count_compact_enc, err_0 := self.db.GetCol(COL_block_entry_count, util.ENC_b_endian_compact_64(block_ordinal))
		if err = err_0; err_0 != nil {
			return
		}
		entry_count = util.DEC_b_endian_compact_64(entry_count_compact_enc)
	}
	if entry_count == 0 {
		return
	}
	block_number_b_endian := util.ENC_b_endian_64(block_ordinal)
	merkle_root_lvl_ordinal := merkle_root_lvl_ordinal(entry_count)
	entry_ordinal := binary.BigEndian.Uint64(found_v[:EntryOrdinalSize])
	proof_bytes := make(Proof, merkle_root_lvl_ordinal+1)
	proof_bytes[0] = append(k, found_v...)
	for lvl_ordinal := MerkleLevelOrdinal(0); lvl_ordinal < merkle_root_lvl_ordinal; lvl_ordinal++ {
		entry_local_pos := entry_ordinal % Arity
		sibling_local_pos := 1 - entry_local_pos
		sibling_ordinal := entry_ordinal - entry_local_pos + sibling_local_pos
		sibling_db_key := merkle_key(lvl_ordinal, util.ENC_b_endian_64(sibling_ordinal), block_number_b_endian)
		found_k, found_v, err_1 := self.db.Find(COL_merkle, sibling_db_key, true)
		if err = err_1; err_1 != nil {
			return
		}
		if !bytes.HasPrefix(found_k, sibling_db_key[:MerkleKeyPrefixSize]) || found_v == nil {
			err = errors.New("Fatal error: no node for merkle key " + string(sibling_db_key))
			return
		}
		proof_bytes[lvl_ordinal+1] = found_v
		entry_ordinal /= Arity
	}
	return proof_bytes, nil
}

var last_block_ordinal_key = []byte("last_block")
var keccak256_0 = crypto.Keccak256(nil)
var b_endian_64_0 = util.ENC_b_endian_64(0)

func merkle_key(lvl MerkleLevelOrdinal, pos []byte, block_num []byte) (key []byte) {
	return append(append(append(make([]byte, 0, MerkleKeySize), lvl), pos...), block_num...)
}

func merkle_root_lvl_ordinal(entry_count EntryOrdinal) MerkleLevelOrdinal {
	return MerkleLevelOrdinal(math.Ceil(math.Log2(float64(entry_count))))
}
