package taraxa_vm

import "github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"

type TaraxaVM struct {
	*base_vm.BaseVM
	TaraxaVMConfig
}
