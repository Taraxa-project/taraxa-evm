package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"
)

type TaraxaVMFactory struct {
	base_vm.BaseVMFactory
	TaraxaVMConfig
}

func (this *TaraxaVMFactory) NewInstance() (ret *TaraxaVM, cleanup func(), err error) {
	var baseVm *base_vm.BaseVM
	baseVm, cleanup, err = this.BaseVMFactory.NewInstance()
	if err != nil {
		return
	}
	ret = &TaraxaVM{BaseVM: baseVm, TaraxaVMConfig: this.TaraxaVMConfig}
	if ret.ConflictDetectorInboxPerTransaction == 0 {
		ret.ConflictDetectorInboxPerTransaction = 5
	}
	util.Assert(ret.NumConcurrentProcesses >= 0)
	util.Assert(ret.ParallelismFactor >= 0)
	if ret.NumConcurrentProcesses == 0 {
		if ret.ParallelismFactor == 0 {
			ret.ParallelismFactor = 1.3
		}
		ret.NumConcurrentProcesses = int(float32(concurrent.CPU_COUNT) * ret.ParallelismFactor)
		if ret.NumConcurrentProcesses < 1 {
			ret.NumConcurrentProcesses = 1
		}
	}
	return
}
