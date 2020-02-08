package main

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/crypto"
)

func main() {
	fmt.Println(crypto.Keccak256(nil))
}
