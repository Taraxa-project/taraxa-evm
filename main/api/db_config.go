package api

import (
	"encoding/json"
	"errors"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/leveldb"
	"github.com/Taraxa-project/taraxa-evm/main/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
)

type DBFactory interface {
	NewDB() (ethdb.Database, error)
}

type memDbFactory struct {
	InitialCapacity int `json:"initialCapacity"`
}

func (this *memDbFactory) NewDB() (ethdb.Database, error) {
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

func (this *GenericDBConfig) NewDB() (ethdb.Database, error) {
	return this.Factory.NewDB()
}

func (this *GenericDBConfig) UnmarshalJSON(b []byte) (err error) {
	var errFatal util.ErrorBarrier
	defer util.Recover(errFatal.Catch(util.SetTo(&err)))
	errFatal.CheckIn(json.Unmarshal(b, &this.TypeConfig))
	switch this.Type {
	case "leveldb":
		this.Factory = new(leveldb.Config)
	case "rocksdb":
		this.Factory = new(rocksdb.Config)
	case "memory":
		this.Factory = new(memDbFactory)
	default:
		return errors.New("Unknown db factory type: " + this.Type)
	}
	errFatal.CheckIn(json.Unmarshal(b, &this.FactoryConfig))
	return
}
