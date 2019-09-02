package ethereum_vm

import "github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"

type EthereumVmFactory struct {
	base_vm.BaseVMFactory
	EthereumVMConfig
}

func (this *EthereumVmFactory) NewInstance() (ret *EthereumVM, cleanup func(), err error) {
	var baseVm *base_vm.BaseVM
	baseVm, cleanup, err = this.BaseVMFactory.NewInstance()
	if err != nil {
		return
	}
	ret = &EthereumVM{BaseVM: baseVm, EthereumVMConfig: this.EthereumVMConfig}
	return
}
