package main

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/block_hash_db"
	"github.com/Taraxa-project/taraxa-evm/main/taraxa_evm"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

func main() {
		str := `
	{"stateRoot": "0x0000000000000000000000000000000000000000000000000000000000000000", "block": {"number": 0, "coinbase": "0x0000000000000000000000000000000000000000", "time": 0, "difficulty": 17179869184, "gasLimit": 5000, "baseRoot": "0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3", "transactions": [], "uncles": []}}
	`
	database := ethdb.NewMemDatabase()
	vm := taraxa_evm.TaraxaVM{
		ExternalApi:   block_hash_db.New(ethdb.NewMemDatabase()),
		Genesis:       core.DefaultGenesisBlock(),
		SourceStateDB: state.NewDatabase(database),
	}
	stateTransition := new(api.StateTransition)
	unmarshalErr := json.Unmarshal([]byte(str), stateTransition)
	util.PanicIfPresent(unmarshalErr)

	ret, stateTransitionErr := vm.TransitionState(stateTransition, &api.ConcurrentSchedule{
		Sequential: nil,
	})
	util.PanicIfPresent(stateTransitionErr)
	fmt.Println(ret.StateRoot.Hex(), "0xd7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544")
	util.Assert(ret.StateRoot.Hex() == "0xd7f8974fb5ac78d9ac099b9ad5018bedc2ce0a72dad1827a1709da30580f0544")
}
