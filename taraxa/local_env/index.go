package main

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
	"github.com/tecbot/gorocksdb"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

func main() {
	db, err := (&rocksdb.Factory{
		File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain",
		ReadOnly:               true,
		Parallelism:            concurrent.CPU_COUNT,
		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
		MaxOpenFiles:           8192,
		OptimizeForPointLookup: 1024,
	}).NewInstance()
	util.PanicIfNotNil(err)
	db1, err11 := (&rocksdb.Factory{
		File:                  "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain_1",
		Parallelism:           concurrent.CPU_COUNT,
		MaxFileOpeningThreads: concurrent.CPU_COUNT,
		MaxBackgroundFlushes:  concurrent.CPU_COUNT,
		MaxOpenFiles:          4000,
		TargetFileSizeBase:    4 * 1024 * 1024,
		WriteBufferSize:       512 * 1024 * 1024,
		UseDirectWrites:       true,
	}).NewInstance()
	util.PanicIfNotNil(err11)
	db1_rocksdb := db1.(*rocksdb.Database).GetDB()
	itr := db1_rocksdb.NewIterator(gorocksdb.NewDefaultReadOptions())
	itr.SeekToLast()
	next_block := 0
	if itr.Valid() {
		if start_block_str := strings.TrimLeft(string(itr.Key().Data()), "0"); len(start_block_str) > 0 {
			last_committed_block, err := strconv.Atoi(start_block_str)
			util.PanicIfNotNil(err)
			next_block = last_committed_block
		}
		next_block++
	}
	for ; ; next_block++ {
		fmt.Println(next_block)
		key := []byte(fmt.Sprintf("%09d", next_block))
		block_json, err := db.Get(key)
		util.PanicIfNotNil(err)
		if block_json == nil {
			break
		}
		block_map := make(map[string]interface{})
		util.PanicIfNotNil(json.Unmarshal(block_json, &block_map))
		block_map["number"] = jsonIntToHex(block_map["number"])
		for _, v := range block_map["transactions"].([]interface{}) {
			tx := v.(map[string]interface{})
			tx["transactionIndex"] = jsonIntToHex(tx["transactionIndex"])
		}
		block_json_1, err12 := json.MarshalIndent(&block_map, "", "  ")
		util.PanicIfNotNil(err12)
		util.PanicIfNotNil(json.Unmarshal(block_json_1, new(vm.Block)))
		util.PanicIfNotNil(db1.Put(key, block_json_1))
	}
	flush_opts := gorocksdb.NewDefaultFlushOptions()
	flush_opts.SetWait(true)
	util.PanicIfNotNil(db1_rocksdb.Flush(flush_opts))
}

func jsonIntToHex(num interface{}) *hexutil.Big {
	hex := new(hexutil.Big)
	hex.ToInt().SetUint64(uint64(num.(float64)))
	return hex
}

func ToBigInt(value interface{}) *big.Int {
	switch value := value.(type) {
	case *big.Int:
		return value
	case []byte:
		return new(big.Int).SetBytes(value)
	case string:
		ret, err := hexutil.DecodeBig(value)
		if err != nil {
			return ret
		}
		ret = new(big.Int)
		util.PanicIfNotNil(ret.UnmarshalText([]byte(value)))
		return ret
	default:
		reflect_val := reflect.ValueOf(value)
		var val_float float64
		if err := util.Try(func() { val_float = reflect_val.Float() }); err == nil {
			if val_float < 0 {
				i := int64(val_float)
				util.Assert(float64(i) == val_float, "Lossy conversion")
				return big.NewInt(int64(val_float))
			}
			i := uint64(val_float)
			util.Assert(float64(i) == val_float, "Lossy conversion")
			return new(big.Int).SetUint64(uint64(val_float))
		}
		var val_uint uint64
		if err := util.Try(func() { val_uint = reflect_val.Uint() }); err == nil {
			return new(big.Int).SetUint64(val_uint)
		}
		var val_int int64
		if err := util.Try(func() { val_int = reflect_val.Int() }); err == nil {
			return big.NewInt(val_int)
		}
	}
	panic(fmt.Sprintf("Could not convert value to bigint: %s", value))
}
