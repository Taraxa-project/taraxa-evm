package taraxa_evm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
)

type StateDB interface {
	vm.StateDB
	Error() error
	Finalise(deleteEmptyObjects bool)
	GetLogs(hash common.Hash) []*types.Log
	OpenTransaction(thash, bhash common.Hash, ti int)
	//CloseTransaction() TODO
	IntermediateRoot(deleteEmptyObjects bool) common.Hash
	Commit(deleteEmptyObjects bool) (root common.Hash, err error)
}
