package ethereum_vm

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/memory"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm"
	"github.com/Taraxa-project/taraxa-evm/taraxa/vm/internal/base_vm"
	"testing"
)

type BlockWithStateRoot = struct {
	*vm.Block
	StateRoot common.Hash `json:"stateRoot"`
}

type EthereumVMIntegrationTest struct {
	StartBlock       uint64
	EndBlock         uint64
	GetBlockByNumber func(uint64) *BlockWithStateRoot
	VMFactory        *EthereumVmFactory
}

func (this *EthereumVMIntegrationTest) Run(t *testing.T) {
	ethereumVM, cleanup, err := this.VMFactory.NewInstance()
	util.PanicIfNotNil(err)
	defer cleanup()
	var prevBlock *BlockWithStateRoot
	if this.StartBlock > 0 {
		prevBlock = this.GetBlockByNumber(this.StartBlock - 1)
	}
	for blockNum := this.StartBlock; blockNum <= this.EndBlock; blockNum++ {
		fmt.Println("block", blockNum)
		block := this.GetBlockByNumber(blockNum)
		stateTransitionRequest := &vm.StateTransitionRequest{Block: block.Block}
		if prevBlock != nil {
			stateTransitionRequest.BaseStateRoot = prevBlock.StateRoot
		}
		result, err := ethereumVM.TransitionState(stateTransitionRequest)
		util.PanicIfNotNil(err)
		util.Assert(result.StateRoot == block.StateRoot, result.StateRoot.Hex(), block.StateRoot.Hex())
		//ethereumVM.CommitToDisk(result.StateRoot)
		prevBlock = block
	}

}

func TestEthereumVMIntegration(t *testing.T) {
	block_db, err := (&rocksdb.Factory{
		File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/blockchain",
		ReadOnly:               true,
		Parallelism:            concurrent.CPU_COUNT,
		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
		MaxOpenFiles:           8192,
		OptimizeForPointLookup: 1024,
	}).NewInstance()
	util.PanicIfNotNil(err)
	getBlockByNumber := func(block_num uint64) *BlockWithStateRoot {
		key := []byte(fmt.Sprintf("%09d", block_num))
		block_json, err := block_db.Get(key)
		util.PanicIfNotNil(err)
		ret := new(BlockWithStateRoot)
		util.PanicIfNotNil(json.Unmarshal(block_json, ret))
		return ret
	}
	factory := new(EthereumVmFactory)
	factory.ReadDBConfig = &base_vm.StateDBConfig{
		DBFactory: &rocksdb.Factory{
			File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/state",
			ReadOnly:               true,
			Parallelism:            concurrent.CPU_COUNT,
			OptimizeForPointLookup: 1024 * 2,
			MaxFileOpeningThreads:  concurrent.CPU_COUNT,
			UseDirectReads:         true,
		},
	}
	factory.WriteDBConfig = &base_vm.StateDBConfig{DBFactory: new(memory.Factory)}
	factory.BlockHashSourceFactory = base_vm.SimpleBlockHashSourceFactory(func(blockNumber uint64) common.Hash {
		return getBlockByNumber(blockNumber).Hash
	})
	test := EthereumVMIntegrationTest{
		StartBlock:       1000000,
		EndBlock:         1050000,
		GetBlockByNumber: getBlockByNumber,
		VMFactory:        factory,
	}
	test.Run(t)
}
