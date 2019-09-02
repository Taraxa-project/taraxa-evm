package base_vm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
)

type VmIOConfig = struct {
	ReadDB  *vm.StateDBConfig  `json:"readDB"`
	WriteDB *vm.StateDBConfig  `json:"writeDB"`
	BlockDB *db.GenericFactory `json:"blockDB"`
}
