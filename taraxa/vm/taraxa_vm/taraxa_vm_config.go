package taraxa_vm

import "github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"

type TaraxaVMConfig struct {
	base_vm.BaseVMConfig
	ConflictDetectorInboxPerTransaction int     `json:"conflictDetectorInboxPerTransaction"`
	NumConcurrentProcesses              int     `json:"numConcurrentProcesses"`
	ParallelismFactor                   float32 `json:"parallelismFactor"`
}
