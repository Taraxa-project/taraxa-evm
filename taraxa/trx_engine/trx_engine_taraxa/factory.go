package trx_engine_taraxa

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/internal/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

type TaraxaTrxEngineFactory struct {
	trx_engine_base.BaseVMFactory
	TaraxaTrxEngineConfig
}

func (this *TaraxaTrxEngineFactory) NewInstance() (ret *TaraxaTrxEngine, cleanup func(), err error) {
	var baseVm *trx_engine_base.BaseVM
	baseVm, cleanup, err = this.BaseVMFactory.NewInstance()
	if err != nil {
		return
	}
	ret = &TaraxaTrxEngine{BaseVM: baseVm, TaraxaTrxEngineConfig: this.TaraxaTrxEngineConfig}
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
