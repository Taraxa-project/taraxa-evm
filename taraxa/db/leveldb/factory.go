package leveldb

import "github.com/Taraxa-project/taraxa-evm/ethdb"

type Factory struct {
	File    string `json:"file"`
	Cache   int    `json:"cache"`
	Handles int    `json:"handles"`
}

func (this *Factory) NewDB() (ethdb.MutableTransactionalDatabase, error) {
	return ethdb.NewLDBDatabase(this.File, this.Cache, this.Handles)
}
