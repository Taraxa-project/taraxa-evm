package ethdb_proxy

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
)

type DatabaseProxy struct {
	ethdb.MutableTransactionalDatabase
	*proxy.BaseProxy
}

func (this *DatabaseProxy) Get(key []byte) (b []byte, e error) {
	defer this.CallDecorator("Get", &key)(&b, &e)
	return this.MutableTransactionalDatabase.Get(key)
}
