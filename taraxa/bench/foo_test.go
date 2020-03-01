package bench

import (
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/rlp"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/tecbot/gorocksdb"
	"math/rand"
	"runtime/debug"
	"testing"
)

func BenchmarkFoo(b *testing.B) {
	debug.SetGCPercent(-1)
	w_ops := gorocksdb.NewDefaultWriteOptions()
	r_ops := gorocksdb.NewDefaultReadOptions()
	rand.Seed(0)
	code_bytes := util.RandomBytes(1024 * 8)
	code_hash := crypto.Keccak256(code_bytes)
	acc_rlp, err_1 := rlp.EncodeToBytes(&state.Account{CodeHash: code_hash})
	util.PanicIfNotNil(err_1)
	acc_addr := util.RandomBytes(20)
	acc_code_key := append(acc_addr, 0)
	acc_code_key_missing_1 := util.RandomBytes(len(acc_code_key))
	acc_code_key_missing_2 := util.RandomBytes(len(acc_code_key))

	new_db := func() *gorocksdb.DB {
		db, err := (&rocksdb.Factory{
			File:           "/tmp/ololololo3",
			UseDirectReads: true,
			BlockCacheSize: 0,
		}).NewInstance()
		util.PanicIfNotNil(err)
		return db.(*rocksdb.Database).GetDB()
	}
	rdb := new_db()
	util.PanicIfNotNil(rdb.Put(w_ops, acc_addr, acc_rlp))
	util.PanicIfNotNil(rdb.Put(w_ops, acc_code_key, code_bytes))
	util.PanicIfNotNil(rdb.Put(w_ops, code_hash, code_bytes))
	util.PanicIfNotNil(rdb.Put(w_ops, acc_code_key_missing_1, code_hash))
	util.PanicIfNotNil(rdb.Delete(w_ops, acc_code_key_missing_2))
	rdb.Close()
	b.Run("new_get_code_best_case", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			rdb := new_db()
			b.StartTimer()
			_, err2 := rdb.GetBytes(r_ops, acc_code_key)
			util.PanicIfNotNil(err2)
			b.StopTimer()
			rdb.Close()
		}
	})
	tmp_acc := new(state.Account)
	b.Run("new_get_code_worst_case_1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			rdb := new_db()
			b.StartTimer()
			_, err := rdb.GetBytes(r_ops, acc_code_key_missing_1)
			util.PanicIfNotNil(err)
			_, err = rdb.GetBytes(r_ops, code_hash)
			util.PanicIfNotNil(err)
			b.StopTimer()
			rdb.Close()
		}
	})
	b.Run("new_get_code_worst_case_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			rdb := new_db()
			b.StartTimer()
			_, err0 := rdb.GetBytes(r_ops, acc_code_key_missing_2)
			util.PanicIfNotNil(err0)
			ret1, err1 := rdb.GetBytes(r_ops, acc_addr)
			util.PanicIfNotNil(err1)
			util.PanicIfNotNil(rlp.DecodeBytes(ret1, tmp_acc))
			_, err2 := rdb.GetBytes(r_ops, code_hash)
			util.PanicIfNotNil(err2)
			b.StopTimer()
			rdb.Close()
		}
	})
	b.Run("current_get_code", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			rdb := new_db()
			b.StartTimer()
			ret1, err1 := rdb.GetBytes(r_ops, acc_addr)
			util.PanicIfNotNil(err1)
			util.PanicIfNotNil(rlp.DecodeBytes(ret1, tmp_acc))
			_, err2 := rdb.GetBytes(r_ops, code_hash)
			util.PanicIfNotNil(err2)
			b.StopTimer()
			rdb.Close()
		}
	})
}
