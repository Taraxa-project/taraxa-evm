package memory

import "github.com/Taraxa-project/taraxa-evm/ethdb"

type Factory struct {
	InitialCapacity int `json:"initialCapacity"`
}

func (this *Factory) NewInstance() (ethdb.MutableTransactionalDatabase, error) {
	return ethdb.NewMemDatabaseWithCap(this.InitialCapacity), nil
}