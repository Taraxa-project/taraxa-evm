package trx_engine_eth

import "github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"

type EthTrxEngineFactory struct {
	trx_engine_base.BaseEngineFactory
	EthTrxEngineConfig
}

func (self *EthTrxEngineFactory) NewInstance() (ret *EthTrxEngine, cleanup func(), err error) {
	var baseVm *trx_engine_base.BaseTrxEngine
	baseVm, cleanup, err = self.BaseEngineFactory.NewInstance()
	if err != nil {
		return
	}
	ret = &EthTrxEngine{BaseTrxEngine: baseVm, EthTrxEngineConfig: self.EthTrxEngineConfig}
	return
}
