package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/taraxa_types"
)

type StateDB interface {
	vm.StateDB
	BeginTransaction(txHash, blockHash common.Hash, txId taraxa_types.TxId)
	CheckPoint(resetDirties bool)
	GetLogs(txHash common.Hash) []*types.Log
	CommitStateChange(deleteEmptyObjects bool) state.StateChange
	Merge(state.StateChange)
	Commit(deleteEmptyObjects bool) (common.Hash, error)
}

type StateDBFactory func() StateDB
type CommitStrategy func(db StateDB) common.Hash

type TransactionResult struct {
	taraxa_types.TxId
	EVMReturnValue []byte
	GasUsed        uint64
	ContractErr    error
	ConsensusErr   error
	Logs           []*types.Log
}

type TransactionResultWithStateChange struct {
	*TransactionResult
	state.StateChange
}
