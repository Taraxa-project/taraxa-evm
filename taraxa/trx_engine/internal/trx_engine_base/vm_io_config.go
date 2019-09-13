package trx_engine_base

import "github.com/Taraxa-project/taraxa-evm/core/vm"

type BlockHashSourceFactory interface {
	NewInstance() (vm.GetHashFunc, error)
}

type SimpleBlockHashSourceFactory vm.GetHashFunc

func (this SimpleBlockHashSourceFactory) NewInstance() (vm.GetHashFunc, error) {
	return vm.GetHashFunc(this), nil
}

type StateDBConfig = struct {
	DBFactory DBFactory `json:"db"`
	CacheSize int       `json:"cacheSize"`
}
