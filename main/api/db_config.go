package api

import (
	"encoding/json"
	"errors"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/cgo_db"
	"github.com/Taraxa-project/taraxa-evm/main/leveldb"
	"github.com/Taraxa-project/taraxa-evm/main/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type DBFactory interface {
	NewDB() (ethdb.MutableTransactionalDatabase, error)
}

var DBFactoryRegistry = map[string]func() DBFactory{
	"leveldb": func() DBFactory {
		return new(leveldb.Config)
	},
	"rocksdb": func() DBFactory {
		return new(rocksdb.Config)
	},
	"memory": func() DBFactory {
		return new(memDbFactory)
	},
	"cgo": func() DBFactory {
		return new(cgo_db.Config)
	},
}

type memDbFactory struct {
	InitialCapacity int `json:"initialCapacity"`
}

func (this *memDbFactory) NewDB() (ethdb.MutableTransactionalDatabase, error) {
	return ethdb.NewMemDatabaseWithCap(this.InitialCapacity), nil
}

type TypeConfig struct {
	Type string `json:"type"`
}

type FactoryConfig struct {
	Factory DBFactory `json:"options"`
}

type GenericDBConfig struct {
	TypeConfig
	FactoryConfig
}

func (this *GenericDBConfig) NewDB() (ethdb.MutableTransactionalDatabase, error) {
	return this.Factory.NewDB()
}

func (this *GenericDBConfig) UnmarshalJSON(b []byte) (err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	errFatal.CheckIn(json.Unmarshal(b, &this.TypeConfig))
	if newFactory, ok := DBFactoryRegistry[this.Type]; ok {
		this.Factory = newFactory()
	} else {
		return errors.New("Unknown db factory type: " + this.Type)
	}
	errFatal.CheckIn(json.Unmarshal(b, &this.FactoryConfig))
	return
}
