package trx_engine_eth

type EthTrxEngineConfig = struct {
	DisableMinerReward bool `json:"disableMinerReward"`
	DisableNonceCheck  bool `json:"disableNonceCheck""`
}
