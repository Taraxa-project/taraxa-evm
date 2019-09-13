package trx_engine_taraxa

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/internal/trx_engine_base"
)

type TaraxaTrxEngine struct {
	*trx_engine_base.BaseVM
	TaraxaTrxEngineConfig
}

func (this *TaraxaTrxEngine) GenerateSchedule(req *trx_engine.StateTransitionRequest) (*trx_engine.ConcurrentSchedule, error) {
	return newScheduleGeneration(this, req).run()
}

func (this *TaraxaTrxEngine) TransitionState(req *trx_engine.StateTransitionRequest, schedule *trx_engine.ConcurrentSchedule) (ret *trx_engine.StateTransitionResult, err error) {
	return
}
