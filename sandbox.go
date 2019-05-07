package main

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/proxy"
	ethdb2 "github.com/Taraxa-project/taraxa-evm/main/proxy/ethdb"
)

type O func()
type ff struct {
	O
}

func main() {
	db := ethdb.NewMemDatabase()
	dbProxy := ethdb2.DatabaseProxy{
		db,
		proxy.Decorators{
			"Get": func(arguments ...proxy.Argument) proxy.ArgumentsCallback {
				fmt.Println("before", arguments)
				return func(arguments ...proxy.Argument) {
					fmt.Println("after", arguments)
				}
			},
		},
	}
	db.Put([]byte{1}, []byte("foo"))
	val, _ := dbProxy.Get([]byte{1})
	fmt.Println(string(val))
}
