package leveldb

import "github.com/Taraxa-project/taraxa-evm/ethdb"

type Config struct {
	File    string `json:"file"`
	Cache   int    `json:"cache"`
	Handles int    `json:"handles"`
}

func (this *Config) NewDB() (ethdb.Database, error) {
	return ethdb.NewLDBDatabase(this.File, this.Cache, this.Handles)
}
