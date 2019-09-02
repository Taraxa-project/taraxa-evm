package base_vm

import (
	"github.com/Taraxa-project/taraxa-evm/core"
)

type BaseVMConfig = struct {
	Genesis *core.Genesis `json:"genesis"`
}
