package facade

import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"testing"
)

func run(requestStr string) *api.Response {
	responseStr := RunJson(requestStr)
	response := new(api.Response)
	json.Unmarshal([]byte(responseStr), response)
	util.PanicOn(response.Error)
	return response
}

func TestFoo(t *testing.T) {
	run(`{"stateTransition": {"stateRoot": "0x0000000000000000000000000000000000000000000000000000000000000000", "block": {"coinbase": "0x0000000000000000000000000000000000000000", "number": "0", "time": "0", "difficulty": "17179869184", "gasLimit": 5000, "hash": "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3"}, "transactions": []}, "stateDatabase": {"file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/tests_perf/out/leveldb/state_db", "cache": 0, "handles": 0}, "blockchainDatabase": {"file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/tests_perf/out/leveldb/blockchain_db", "cache": 0, "handles": 0}, "concurrentSchedule": {"sequential": []}}`)
	run(`{"stateTransition": {"stateRoot": "0xd7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544", "block": {"coinbase": "0x05a56e2d52c817161883f50c441c3228cfe54d9f", "number": "1", "time": "1438269988", "difficulty": "17171480576", "gasLimit": 5000, "hash": "0x88e96d4537bea4d9c05d12549907b32561d3bf31f45aae734cdc119f13406cb6"}, "transactions": []}, "stateDatabase": {"file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/tests_perf/out/leveldb/state_db", "cache": 0, "handles": 0}, "blockchainDatabase": {"file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/tests_perf/out/leveldb/blockchain_db", "cache": 0, "handles": 0}, "concurrentSchedule": {"sequential": []}}`)
	run(`{"stateTransition": {"stateRoot": "0xd67e4d450343046425ae4271474353857ab860dbc0a1dde64b41b5cd3a532bf3", "block": {"coinbase": "0xdd2f1e6e498202e86d8f5442af596580a4f03c2c", "number": "2", "time": "1438270017", "difficulty": "17163096064", "gasLimit": 5000, "hash": "0xb495a1d7e6663152ae92708da4843337b958146015a2802f4193a410044698c9"}, "transactions": []}, "stateDatabase": {"file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/tests_perf/out/leveldb/state_db", "cache": 0, "handles": 0}, "blockchainDatabase": {"file": "/Users/compuktor/projects/taraxa.io/taraxa-evm/tests_perf/out/leveldb/blockchain_db", "cache": 0, "handles": 0}, "concurrentSchedule": {"sequential": []}}`)
}
