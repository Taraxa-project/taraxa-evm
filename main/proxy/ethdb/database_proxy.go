package ethdb

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
)

type DatabaseProxy struct {
	ethdb.Database
	proxy.Decorators
}

func (this *DatabaseProxy) Get(key []byte) (b []byte, e error) {
	after := this.BeforeCall("Get", &key)
	defer after(&b, &e)
	return this.Database.Get(key)
}

func (this *DatabaseProxy) Has(key []byte) (b bool, e error) {
	after := this.BeforeCall("Has", &key)
	defer after(&b, &e)
	return this.Database.Has(key)
}
