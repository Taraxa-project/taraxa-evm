package ethdb_proxy

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
)

type DatabaseProxy struct {
	ethdb.Database
	*proxy.BaseProxy
}

func (this *DatabaseProxy) Get(key []byte) (b []byte, e error) {
	defer this.CallDecorator("Get", &key)(&b, &e)
	return this.Database.Get(key)
}

func (this *DatabaseProxy) Has(key []byte) (b bool, e error) {
	defer this.CallDecorator("Has", &key)(&b, &e)
	return this.Database.Has(key)
}
