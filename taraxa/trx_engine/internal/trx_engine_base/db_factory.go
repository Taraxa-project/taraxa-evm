package trx_engine_base

import (
	"encoding/json"
	"errors"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/cgo"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/leveldb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/memory"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
)

type DBFactory interface {
	NewInstance() (ethdb.MutableTransactionalDatabase, error)
}

var FactoryRegistry = map[string]func() DBFactory{
	"leveldb": func() DBFactory {
		return new(leveldb.Factory)
	},
	"rocksdb": func() DBFactory {
		return new(rocksdb.Factory)
	},
	"memory": func() DBFactory {
		return new(memory.Factory)
	},
	"cgo": func() DBFactory {
		return new(cgo.Factory)
	},
}

type FactoryType = struct {
	Type string `json:"type"`
}

type FactoryOptions = struct {
	Factory DBFactory `json:"options"`
}

type GenericFactory struct {
	FactoryType
	FactoryOptions
}

func (this *GenericFactory) NewInstance() (ethdb.MutableTransactionalDatabase, error) {
	return this.Factory.NewInstance()
}

func (this *GenericFactory) UnmarshalJSON(b []byte) (err error) {
	var errFatal concurrent.AtomicError
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	errFatal.SetOrPanicIfPresent(json.Unmarshal(b, &this.FactoryType))
	if newFactory, ok := FactoryRegistry[this.Type]; ok {
		this.Factory = newFactory()
	} else {
		return errors.New("Unknown db factory type: " + this.Type)
	}
	errFatal.SetOrPanicIfPresent(json.Unmarshal(b, &this.FactoryOptions))
	return
}
