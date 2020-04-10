package vm

import "github.com/Taraxa-project/taraxa-evm/common"

type LogRecord struct {
	Address common.Address
	Topics  []common.Hash
	Data    []byte
}
