package ethereum_vm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm/taraxa_vm"
	"testing"
)

func TestFoo(t *testing.T) {

}

type TestContext struct {
	GetBlockByNumber func(uint64) *vm.Block
	VMFactory        taraxa_vm.TaraxaVMFactory
}

func test(ctx *TestContext) {

}
