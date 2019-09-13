package trx_engine_taraxa

type TaraxaTrxEngineConfig struct {
	ConflictDetectorInboxPerTransaction int     `json:"conflictDetectorInboxPerTransaction"`
	NumConcurrentProcesses              int     `json:"numConcurrentProcesses"`
	ParallelismFactor                   float32 `json:"parallelismFactor"`
}
