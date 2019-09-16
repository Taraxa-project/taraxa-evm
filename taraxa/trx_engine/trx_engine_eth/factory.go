package trx_engine_eth

import "github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"

type EthTrxEngineFactory struct {
	trx_engine_base.BaseVMFactory
	EthTrxEngineConfig
}

func (this *EthTrxEngineFactory) NewInstance() (ret *EthTrxEngine, cleanup func(), err error) {
	var baseVm *trx_engine_base.BaseTrxEngine
	baseVm, cleanup, err = this.BaseVMFactory.NewInstance()
	if err != nil {
		return
	}
	ret = &EthTrxEngine{BaseTrxEngine: baseVm, EthTrxEngineConfig: this.EthTrxEngineConfig}
	return
}
