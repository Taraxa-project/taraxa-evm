package tests

import (
	"bytes"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state/eth_trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state/experimental_state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestRoot(t *testing.T) {
	targets := make(map[string]func(t *testing.T) state.State)
	TARGET := func(name string, prepare func(t *testing.T) state.State) {
		targets[name] = prepare
	}
	TEST := func(name string, test func(t *testing.T, s state.State)) {
		for target_name, target_factory := range targets {
			t.Run(name+"/"+target_name, func(t *testing.T) {
				test(t, target_factory(t))
			})
		}
	}

	TARGET_DISABLED("trie_state", func(t *testing.T) state.State {
		path := os.TempDir() + strings.ReplaceAll(t.Name(), "/", "_")
		util.PanicIfNotNil(os.RemoveAll(path))
		target, err := eth_trie.NewTrieState(&eth_trie.TrieStateConfig{
			RocksDBExtDBConfig: rocksdb_ext.RocksDBExtDBConfig{
				Path: path,
			},
		})
		util.PanicIfNotNil(err)
		return target
	})
	TARGET("experimental_state", func(t *testing.T) state.State {
		path := os.TempDir() + strings.ReplaceAll(t.Name(), "/", "_")
		util.PanicIfNotNil(os.RemoveAll(path))
		target, err := experimental_state.NewExperimentalState(&experimental_state.ExperimentalStateConfig{
			RocksDBExtDBConfig: rocksdb_ext.RocksDBExtDBConfig{
				Path: path,
			},
		})
		util.PanicIfNotNil(err)
		return target
	})

	test_entry_cnt := 100000
	keys := make([][]byte, test_entry_cnt)
	for i := range keys {
		keys[i] = random_bytes("key", i)
	}
	values := make([][]byte, test_entry_cnt)
	for i := range keys {
		values[i] = random_bytes("value", i)
	}

	TEST_DISABLED("test_1", func(t *testing.T, s state.State) {
		block_num, checksum, err_1 := s.CommitBlock(state.StateChange{
			state.StateEntry{
				keys[0],
				values[0],
			},
			state.StateEntry{
				keys[1],
				values[1],
			},
		})
		util.PanicIfNotNil(err_1)
		v, err_2 := s.Get(block_num, keys[0])
		util.PanicIfNotNil(err_2)
		util.Assert(bytes.Compare(v, values[0]) == 0)
		proof, err_3 := s.GetWithProof(block_num, keys[1])
		util.PanicIfNotNil(err_3)
		v_proven, err_4 := proof.Verify(checksum, keys[1])
		util.PanicIfNotNil(err_4)
		util.Assert(bytes.Compare(v_proven, values[1]) == 0)
	})
	TEST("test_2", func(t *testing.T, s state.State) {
		for i := 0; i < 15; i++ {
			fmt.Println(i)
			used_keys := make(map[string]bool)
			state_change := make(state.StateChange, int((float64(rand.Int()%40)/100)*float64(test_entry_cnt)))
			for i := range state_change {
				for {
					key := keys[rand.Int()%test_entry_cnt]
					if !used_keys[string(key)] {
						used_keys[string(key)] = true
						state_change[i] = state.StateEntry{key, values[rand.Int()%test_entry_cnt]}
						break
					}
				}
			}
			fmt.Printf("entry count: %s\n", strconv.Itoa(len(state_change)))
			block_ordinal, digest, err_0 := s.CommitBlock(state_change)
			util.PanicIfNotNil(err_0)
			NOOP(block_ordinal, digest)
			for _, entry := range state_change {
				v, err_1 := s.Get(block_ordinal, entry.K)
				util.PanicIfNotNil(err_1)
				util.Assert(bytes.Compare(v, entry.V) == 0)
				proof, err_2 := s.GetWithProof(block_ordinal, entry.K)
				util.PanicIfNotNil(err_2)
				v_proven, err_3 := proof.Verify(digest, entry.K)
				util.PanicIfNotNil(err_3)
				util.Assert(bytes.Compare(v_proven, entry.V) == 0)
			}
		}
	})
}

func random_bytes(tag string, id int) (ret []byte) {
	ret = append(ret, util.ENC_b_endian_64(rand.Uint64())...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = copy_bytes(ret[:2])
	return
}

func NOOP(...interface{}) {

}

var TEST_DISABLED = NOOP
var TARGET_DISABLED = NOOP

func copy_bytes(src []byte) []byte {
	return append(make([]byte, 0, len(src)), src...)
}
