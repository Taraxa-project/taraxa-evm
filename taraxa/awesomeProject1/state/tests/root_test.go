package tests

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state/eth_trie"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/state/experimental_state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util/rocksdb_ext"
	"github.com/Taraxa-project/taraxa-evm/taraxa/test_util"
	"github.com/emicklei/dot"
	"github.com/tecbot/gorocksdb"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func BenchmarkRoot(ctx *testing.B) {
	test_util.RunTests(ctx, Root)
}

func TestRoot(ctx *testing.T) {
	test_util.RunTests(ctx, Root)
}

func Root(ctx *test_util.TestContext) {
	rand_state_change := func(size int) state.StateChange {
		ret := make(state.StateChange, 0, size)
		used_keys := make(map[string]bool)
		for len(ret) < cap(ret) {
			k := random_bytes()
			if !used_keys[string(k)] {
				used_keys[string(k)] = true
				ret = append(ret, state.StateEntry{k, random_bytes()})
			}
		}
		return ret
	}
	rand_state_change_sortition := func(state_change state.StateChange, ratio float64) state.StateChange {
		ret := make(state.StateChange, 0, int(float64(len(state_change))*ratio))
		used_positions := make(map[int]bool)
		for len(ret) < cap(ret) {
			pos := rand.Int() % len(state_change)
			if !used_positions[pos] {
				used_positions[pos] = true
				ret = append(ret, state.StateEntry{state_change[pos].K, state_change[rand.Int()%len(state_change)].V})
			}
		}
		return ret
	}
	test_entry_cnt := 100000
	test_state_change := rand_state_change(test_entry_cnt)

	new_trie_state := func(ctx testing.TB) state.State {
		path := os.TempDir() + strings.ReplaceAll(ctx.Name(), "/", "_")
		util.PanicIfNotNil(os.RemoveAll(path))
		target, err := eth_trie.NewTrieState(&eth_trie.TrieStateConfig{
			RocksDBExtDBConfig: rocksdb_ext.RocksDBExtDBConfig{
				Path: path,
			},
		})
		util.PanicIfNotNil(err)
		return target
	}
	new_experimental_state := func(ctx testing.TB) state.State {
		path := os.TempDir() + strings.ReplaceAll(ctx.Name(), "/", "_")
		util.PanicIfNotNil(os.RemoveAll(path))
		target, err := experimental_state.NewExperimentalState(&experimental_state.ExperimentalStateConfig{
			RocksDBExtDBConfig: rocksdb_ext.RocksDBExtDBConfig{
				Path: path,
			},
		})
		util.PanicIfNotNil(err)
		return target
	}
	ctx.TARGET_TEST("trie_state", new_trie_state)
	ctx.TARGET_TEST("experimental_state", new_experimental_state)
	ctx.TEST("rand_test", func(ctx *testing.T, s state.State) {
		ctx.Skip()
		state_change := test_state_change
		for i := 0; i < 5; i++ {
			fmt.Println(i)
			fmt.Println("entry cnt:", len(state_change))
			block_ordinal, digest, err_0 := s.CommitBlock(state_change)
			util.PanicIfNotNil(err_0)
			NOOP(block_ordinal, digest)
			//for _, entry := range state_change {
			//	v, err_1 := s.Get(block_ordinal, entry.K)
			//	util.PanicIfNotNil(err_1)
			//	util.Assert(bytes.Compare(v, entry.V) == 0)
			//	proof, err_2 := s.GetWithProof(block_ordinal, entry.K)
			//	util.PanicIfNotNil(err_2)
			//	v_proven, err_3 := proof.Verify(digest, entry.K)
			//	util.PanicIfNotNil(err_3)
			//	util.Assert(bytes.Compare(v_proven, entry.V) == 0)
			//}
			state_change = rand_state_change_sortition(test_state_change, 0.001)
		}
	})
	ctx.TEST("foo", func(t *testing.T) {
		s := new_trie_state(t).(*eth_trie.TrieState)
		_, _, err_0 := s.CommitBlock(test_state_change)
		util.PanicIfNotNil(err_0)
		//s.Dot_g = dot.NewGraph()
		//s.Dot_g.Attr("ratio", "0.5")
		for i := 0; i < 10; i++ {
			fmt.Println(i)
			_, _, err_1 := s.CommitBlock(rand_state_change_sortition(test_state_change, 0.001))
			util.PanicIfNotNil(err_1)
		}
		//view(s.Dot_g)
	})
	ctx.BENCH("trie_state_rand_prepop", func(ctx *testing.B) {
		ctx.Skip()
		ctx.StopTimer()
		sut := new_trie_state(ctx)
		defer sut.Close()
		state_change := test_state_change
		_, _, err_0 := sut.CommitBlock(state_change)
		util.PanicIfNotNil(err_0)
		for i := 0; i < ctx.N; i++ {
			sc := rand_state_change_sortition(state_change, 0.0001)
			ctx.StartTimer()
			sut.CommitBlock(sc)
			ctx.StopTimer()
		}
	})
	ctx.BENCH("exp_state_rand_prepop", func(ctx *testing.B) {
		ctx.Skip()
		ctx.StopTimer()
		sut := new_experimental_state(ctx)
		defer sut.Close()
		state_change := test_state_change
		_, _, err_0 := sut.CommitBlock(state_change)
		util.PanicIfNotNil(err_0)
		ctx.ResetTimer()
		for i := 0; i < ctx.N; i++ {
			sc := rand_state_change_sortition(state_change, 0.0001)
			ctx.StartTimer()
			sut.CommitBlock(sc)
			ctx.StopTimer()
		}
	})

	func() {
		db_cfg := new(rocksdb_ext.RocksDBExtConfig)
		db_cfg.Path = os.TempDir() + string(os.PathSeparator) + "rocksdb_benchmarks"
		//os.RemoveAll(db_cfg.Path)
		db, err_0 := rocksdb_ext.NewRocksDBExt(db_cfg)
		util.PanicIfNotNil(err_0)
		//for _, e := range test_state_change {
		//	util.PanicIfNotNil(db.Put(rocksdb_ext.Default_opts_w, e.K, e.V))
		//}
		db.Close()
		ctx.BENCH("db_get", func(ctx *testing.B) {
			db, err_0 := rocksdb_ext.NewRocksDBExt(db_cfg)
			util.PanicIfNotNil(err_0)
			defer db.Close()
			ctx.ResetTimer()
			for i := 0; i < ctx.N; i++ {
				_, err_1 := db.Get(rocksdb_ext.Default_opts_r, test_state_change[i%len(test_state_change)].K)
				util.PanicIfNotNil(err_1)
			}
		})
		ctx.BENCH("db_seek", func(ctx *testing.B) {
			db, err_0 := rocksdb_ext.NewRocksDBExt(db_cfg)
			util.PanicIfNotNil(err_0)
			defer db.Close()
			opts := gorocksdb.NewDefaultReadOptions()
			//opts.SetReadaheadSize(1024 * 1024 * 1024 * 2)
			//opts.SetTailing(true)
			opts.SetPinData(true)
			itr := db.NewIterator(opts)
			ctx.ResetTimer()
			for i := 0; i < ctx.N; i++ {
				itr.Seek(test_state_change[i%len(test_state_change)].K)
				k_s, v_s := itr.Key(), itr.Value()
				defer k_s.Free()
				defer v_s.Free()
				NOOP(copy_bytes(k_s.Data()), copy_bytes(v_s.Data()))
			}
		})
	}()
}

func random_bytes() (ret []byte) {
	//ret = append(ret, util.ENC_b_endian_64(rand.Uint64())...)
	ret = append(ret, crypto.Keccak256(
		util.ENC_b_endian_64(rand.Uint64()),
		util.ENC_b_endian_64(rand.Uint64()),
		util.ENC_b_endian_64(rand.Uint64()),
		util.ENC_b_endian_64(rand.Uint64()),
	)...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = append(ret, crypto.Keccak256(util.ENC_b_endian_64(rand.Uint64()))...)
	//ret = copy_bytes(ret[:2])
	//ret = util.ENC_b_endian_64(rand.Uint64())
	//return util.ENC_b_endian_64(rand.Uint64())
	return
}

func NOOP(...interface{}) {

}

func copy_bytes(src []byte) []byte {
	return append(make([]byte, 0, len(src)), src...)
}

func view(g *dot.Graph) {
	dot_f_path := "/tmp/tmp.dot"
	pdf_f_path := "/tmp/tmp.pdf"
	os.Remove(dot_f_path)
	os.Remove(pdf_f_path)
	dot_f, err_0 := os.Create(dot_f_path)
	util.PanicIfNotNil(err_0)
	g.Write(dot_f)
	util.PanicIfNotNil(exec.Command("dot", "-Tpdf", dot_f_path, "-o", pdf_f_path).Run())
	util.PanicIfNotNil(exec.Command("open", pdf_f_path).Run())
}
