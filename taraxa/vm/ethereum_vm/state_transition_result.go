package ethereum_vm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
)

type StateTransitionResult = struct {
	Result vm.StateTransitionResult `json:"result"`
	Error  error                    `json:"error"`
}
