package experimental_state

import (
	"encoding/binary"
	"fmt"
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
	block_cnt            state.BlockOrdinal
	entry_cnt            EntryOrdinal
	absent_node_hash_gen AbsentNodeHashGenerator
}
type ExperimentalStateConfig struct {
	rocksdb_ext.RocksDBExtDBConfig
	ColumnOpts           [COL_cnt]rocksdb_ext.RocksDBExtColumnOpts
	absent_node_hash_gen AbsentNodeHashGenerator
}
type MerkleLevelOrdinal = byte
type EntryOrdinal = uint64
type MerkleLevel = map[EntryOrdinal][]byte
type AbsentNodeHashGenerator = func(MerkleLevelOrdinal, EntryOrdinal, state.BlockOrdinal) []byte

const (
	Arity                  = 2
	BlockOrdinalSize       = int(unsafe.Sizeof(state.BlockOrdinal(0)))
	EntryOrdinalSize       = int(unsafe.Sizeof(EntryOrdinal(0)))
	MerkleLevelOrdinalSize = int(unsafe.Sizeof(MerkleLevelOrdinal(0)))
	MerkleKeyPrefixSize    = MerkleLevelOrdinalSize + EntryOrdinalSize
	MerkleKeySize          = MerkleLevelOrdinalSize + EntryOrdinalSize + BlockOrdinalSize
)

type Column = int

const (
	COL_default Column = iota
	COL_merkle
	COL_merkle_historical
	COL_entries
	COL_entries_historical
	COL_block_entry_cnt
	COL_cnt
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
	self.absent_node_hash_gen = cfg.absent_node_hash_gen
	if self.absent_node_hash_gen == nil {
		self.absent_node_hash_gen = func(MerkleLevelOrdinal, EntryOrdinal, state.BlockOrdinal) []byte {
			return keccak256_0
		}
	}
	block_ord_enc_compact, err_0 := self.db.GetCol(COL_default, last_block_ord_key)
	if err = err_0; err_0 != nil || block_ord_enc_compact == nil {
		return
	}
	self.block_cnt = util.DEC_b_endian_compact_64(block_ord_enc_compact) + 1
	entry_cnt_enc_compact, err_1 := self.db.GetCol(COL_block_entry_cnt, block_ord_enc_compact)
	if err = err_1; err_1 != nil {
		return
	}
	self.entry_cnt = util.DEC_b_endian_compact_64(entry_cnt_enc_compact)
	return
}

// TODO state_change order deterministically defined by the caller layers, possibly VRF
// TODO defragmentation sortition could be deterministically defined by the caller, possibly VRF
// TODO empty (new subtree after the doubling) merkle node hashes deterministically defined by the caller
// TODO use hashed key, possibly store preimages
// TODO fixed size value optimizations
// TODO Consider static ordinal calculation e.g. mod

// TODO reuse buffers more
// TODO parallelism

// TODO GetForUpdate
// TODO configurable arity
func (self *ExperimentalState) CommitBlock(state_change state.StateChange) (block_ord state.BlockOrdinal, digest []byte, err error) {
	block_ord = self.block_cnt
	block_ord_enc_compact := util.ENC_b_endian_compact_64(block_ord)
	if len(state_change) == 0 {
		digest = keccak256_0
		merkle_root_key := merkle_key(merkle_root_lvl_ord(self.entry_cnt), 0)
		if digest, err = self.db.GetCol(COL_merkle, merkle_root_key); err != nil {
			return
		}
		if err = self.db.PutCol(COL_default, last_block_ord_key, block_ord_enc_compact); err == nil {
			self.block_cnt++
		}
		return
	}
	entry_cnt := self.entry_cnt
	//block_ord_b_endian := util.ENC_b_endian_64(block_ord)
	merkle_lvl_ord := MerkleLevelOrdinal(0)
	merkle_lvl := make(MerkleLevel, len(state_change))
	batch := gorocksdb.NewWriteBatch()
	for _, entry := range state_change {
		//fmt.Println("entry:", util.BytesToStrPadded(entry.K), "||", util.BytesToStrPadded(entry.V))
		//self.db.ToggleProfiling()
		found_v, err_0 := self.db.GetCol(COL_entries, entry.K)
		//self.db.ToggleProfiling()
		if err = err_0; err_0 != nil {
			return
		}
		var entry_ord EntryOrdinal
		var entry_ord_b_endian []byte
		if found_v != nil {
			entry_ord_b_endian = found_v[:EntryOrdinalSize]
			entry_ord = binary.BigEndian.Uint64(entry_ord_b_endian)
		} else {
			entry_ord = entry_cnt
			entry_ord_b_endian = util.ENC_b_endian_64(entry_ord)
			entry_cnt++
		}
		entry_v := append(entry_ord_b_endian, entry.V...)
		//self.db.ToggleProfiling()
		self.db.BatchPutCol(batch, COL_entries, entry.K, entry_v)
		//self.db.ToggleProfiling()
		//self.db.BatchPutCol(batch, COL_entries_historical, append(entry.K, block_ord_b_endian...), entry_v)
		entry_hash := crypto.Keccak256(entry.K, entry_v)
		//self.db.ToggleProfiling()
		self.db.BatchPutCol(batch, COL_merkle, merkle_key(merkle_lvl_ord, entry_ord), entry_hash)
		//self.db.ToggleProfiling()
		//self.db.BatchPutCol(batch, COL_merkle_historical, merkle_key_historical(merkle_lvl_ord, entry_ord_b_endian, block_ord_b_endian), entry_hash)

		//fmt.Printf("New node pos: %s, key: [ %s ], hash: [ %s ]\n",
		//	strconv.Itoa(int(entry_ord)), util.BytesToStrPadded(entry.K), util.BytesToStrPadded(entry_hash))

		merkle_lvl[entry_ord] = entry_hash
	}
	sibling_hashes_buf := make([][]byte, Arity)
	merkle_root_lvl_ord := merkle_root_lvl_ord(entry_cnt)
	curr_lvl_entry_cnt := self.entry_cnt
	for ; merkle_lvl_ord < merkle_root_lvl_ord; merkle_lvl_ord++ {
		// TODO precise allocation
		next_lvl := make(MerkleLevel)
		next_lvl_ord := merkle_lvl_ord + 1
		for entry_ord, hash := range merkle_lvl {
			//delete(merkle_lvl, entry_ord)
			entry_local_pos := entry_ord % Arity
			sibling_local_pos := 1 - entry_local_pos
			sibling_ord := entry_ord - entry_local_pos + sibling_local_pos
			sibling_hash, ok := merkle_lvl[sibling_ord]
			if ok {
				delete(merkle_lvl, sibling_ord)
			} else if sibling_ord < curr_lvl_entry_cnt {
				sibling_db_key := merkle_key(merkle_lvl_ord, sibling_ord)
				if sibling_hash, err = self.db.GetCol(COL_merkle, sibling_db_key); err != nil {
					return
				}
				util.Assert(sibling_hash != nil)
			} else {
				if sibling_hash, err = self.absent_node_hash(merkle_lvl_ord, sibling_ord, block_ord); err != nil {
					// TODO don't store empty hash
					//self.db.BatchPutCol(batch, COL_merkle, sibling_db_key, sibling_hash)
					//self.db.BatchPutCol(batch, COL_merkle_historical, merkle_key_historical(merkle_lvl_ord, util.ENC_b_endian_64(sibling_ord), block_ord_b_endian), sibling_hash)
					return
				}
			}
			sibling_hashes_buf[entry_local_pos] = hash
			sibling_hashes_buf[sibling_local_pos] = sibling_hash
			parent_hash := crypto.Keccak256(sibling_hashes_buf...)
			parent_ordnal := entry_ord / Arity
			self.db.BatchPutCol(batch, COL_merkle, merkle_key(next_lvl_ord, parent_ordnal), parent_hash)
			//fmt.Printf(
			//	"Hashed nodes at lvl %s:\n"+
			//		"  pos: %s, value: [ %s ]\n"+
			//		"  pos: %s, value: [ %s ]\n"+
			//		"  -> pos: %s, value: [ %s ]\n",
			//	strconv.Itoa(int(merkle_lvl_ord)),
			//	strconv.Itoa(int(entry_ord)), util.BytesToStrPadded(hash),
			//	strconv.Itoa(int(sibling_ord)), util.BytesToStrPadded(sibling_hash),
			//	strconv.Itoa(int(parent_ordnal)), util.BytesToStrPadded(parent_hash),
			//)
			//self.db.BatchPutCol(batch, COL_merkle_historical, merkle_key_historical(next_lvl_ord, util.ENC_b_endian_64(parent_ordnal), block_ord_b_endian), parent_hash)
			next_lvl[parent_ordnal] = parent_hash
		}
		curr_lvl_entry_cnt = EntryOrdinal(math.Ceil(float64(curr_lvl_entry_cnt) / Arity))
		merkle_lvl = next_lvl
	}
	digest = merkle_lvl[0]
	self.db.BatchPutCol(batch, COL_block_entry_cnt, block_ord_enc_compact, util.ENC_b_endian_compact_64(entry_cnt))
	self.db.BatchPutCol(batch, COL_default, last_block_ord_key, block_ord_enc_compact)
	if err = self.db.Commit(batch); err != nil {
		return
	}
	self.entry_cnt = entry_cnt
	self.block_cnt++
	//self.db.Dump()
	return
}

func (self *ExperimentalState) Get(block_ord state.BlockOrdinal, k []byte) (ret []byte, err error) {
	ret, err = self.getRaw(block_ord, k)
	if ret != nil && err == nil {
		ret = ret[EntryOrdinalSize:]
	}
	return
}

func (self *ExperimentalState) GetWithProof(block_ord state.BlockOrdinal, k []byte) (proof state.ValueProof, err error) {
	v, err_0 := self.getRaw(block_ord, k)
	if err = err_0; err_0 != nil || v == nil {
		return
	}
	entry_cnt := self.entry_cnt
	historical := self.block_cnt-1 != block_ord
	if historical {
		entry_cnt_compact_enc, err_0 := self.db.GetCol(COL_block_entry_cnt, util.ENC_b_endian_compact_64(block_ord))
		if err = err_0; err_0 != nil {
			return
		}
		entry_cnt = util.DEC_b_endian_compact_64(entry_cnt_compact_enc)
	}
	merkle_root_lvl_ord := merkle_root_lvl_ord(entry_cnt)
	proof_bytes := make(Proof, merkle_root_lvl_ord+1)
	proof_bytes[0] = append(k, v...)
	entry_ord := binary.BigEndian.Uint64(v[:EntryOrdinalSize])
	curr_lvl_entry_cnt := entry_cnt
	//fmt.Println("Proving for: [", util.BytesToStrPadded(k), "] ordinal:", entry_ord)
	for lvl_ord := MerkleLevelOrdinal(0); lvl_ord < merkle_root_lvl_ord; lvl_ord++ {
		entry_local_pos := entry_ord % Arity
		sibling_local_pos := 1 - entry_local_pos
		sibling_ord := entry_ord - entry_local_pos + sibling_local_pos
		var sibling_v []byte
		if historical {
			// TODO reuse blk num bytes
			block_number_b_endian := util.ENC_b_endian_64(block_ord)
			sibling_db_key := merkle_key_historical(lvl_ord, util.ENC_b_endian_64(sibling_ord), block_number_b_endian)
			sibling_v, err = self.db.MaxForPrefix(COL_merkle_historical, sibling_db_key, MerkleKeyPrefixSize)
		} else if sibling_ord < curr_lvl_entry_cnt {
			sibling_v, err = self.db.GetCol(COL_merkle, merkle_key(lvl_ord, sibling_ord))
		} else {
			sibling_v, err = self.absent_node_hash(lvl_ord, sibling_ord, block_ord)
		}
		if err != nil {
			return
		}
		if sibling_v == nil {
			err = fmt.Errorf(
				"Fatal error: no node for merkle key (level: %s, pos: %s, block: %s)",
				lvl_ord, sibling_ord, block_ord,
			)
			return
		}
		//fmt.Println("proof node at lvl", lvl_ord, "pos:", sibling_ord, "value:", util.BytesToStrPadded(sibling_v))
		proof_bytes[lvl_ord+1] = sibling_v
		entry_ord /= Arity
		curr_lvl_entry_cnt = EntryOrdinal(math.Ceil(float64(curr_lvl_entry_cnt) / Arity))
	}
	return proof_bytes, nil
}

func (self *ExperimentalState) absent_node_hash(lvl MerkleLevelOrdinal, pos EntryOrdinal, blk state.BlockOrdinal) (ret []byte, err error) {
	ret = self.absent_node_hash_gen(lvl, pos, blk)
	if len(keccak256_0) != len(ret) {
		err = fmt.Errorf("absent_node_hash_gen must return %s-byte array", len(keccak256_0))
	}
	return
}

func (self *ExperimentalState) getRaw(block_ord state.BlockOrdinal, k []byte) (ret []byte, err error) {
	if self.block_cnt <= block_ord {
		return
	}
	if self.block_cnt-1 == block_ord {
		return self.db.GetCol(COL_entries, k)
	}
	return self.db.MaxForPrefix(COL_entries_historical, append(k, util.ENC_b_endian_64(block_ord)...), len(k))
}

func (self *ExperimentalState) Close() {
	self.db.Close()
}

var last_block_ord_key = []byte("last_block")
var keccak256_0 = crypto.Keccak256(nil)

func merkle_key(lvl_ord MerkleLevelOrdinal, entry_ord EntryOrdinal) []byte {
	return util.ENC_b_endian_compact_64_w_alloc(entry_ord, func(i int) []byte {
		return append(make([]byte, 0, i+1), lvl_ord)
	})
}
func merkle_key_historical(lvl MerkleLevelOrdinal, pos []byte, block_num []byte) (key []byte) {
	return append(append(append(make([]byte, 0, MerkleKeySize), lvl), pos...), block_num...)
}

func merkle_root_lvl_ord(entry_cnt EntryOrdinal) MerkleLevelOrdinal {
	return MerkleLevelOrdinal(math.Ceil(math.Log2(float64(entry_cnt))))
}
