package main

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/metric_utils"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/ethdb_proxy"
	"github.com/Taraxa-project/taraxa-evm/main/proxy/state_db_proxy"
)

type O func()

type foo struct {
	i int
	s string
}

type Foo func()

func (this Foo) call() {
	fmt.Println("foo")
}

func main() {
	var foo metric_utils.AtomicCounter
	db := ethdb.NewMemDatabase()
	readDiskDBPRoxy := ethdb_proxy.DatabaseProxy{Database: db}
	dbProxy := state_db_proxy.DatabaseProxy{Database: state.NewDatabase(readDiskDBPRoxy)}
	readDiskDBPRoxy.Decorators.Register("Get", metric_utils.MeasureElapsedTime(&foo))
	readDiskDBPRoxy.Decorators.Register("Has", metric_utils.MeasureElapsedTime(&foo))
	dbProxy.Decorators.Register("OpenTrie", func(arguments ...proxy.Argument) proxy.ArgumentsCallback {
		fmt.Println("foo")
		return func(arguments ...proxy.Argument) {
			fmt.Println("bar")
		}
	})
	readDiskDBPRoxy.Get(nil)
	dbProxy.OpenTrie(common.Hash{})
	fmt.Println(foo)
}
