package trx_engine_base

import (
	"github.com/Taraxa-project/taraxa-evm/core"
)

type BaseVMConfig = struct {
	Genesis *core.Genesis `json:"genesis"`
}
