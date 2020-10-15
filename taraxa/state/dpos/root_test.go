package dpos

import (
	"fmt"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/rlp"
)

func TestFoo(t *testing.T) {
	s := "0xdbda94d5ee1a7f987763899258d26f1477f9c8d93e19e5c482271080"
	foo := make(InboundTransfers)
	rlp.MustDecodeBytes(common.Hex2Bytes(s), &foo)
	fmt.Println(foo)
}
