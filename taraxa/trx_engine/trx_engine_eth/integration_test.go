package trx_engine_eth

import (
	"encoding/json"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/db/rocksdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"os"
	"testing"
)

type BlockWithStateRoot = struct {
	*trx_engine.Block
	StateRoot common.Hash `json:"stateRoot"`
}

type EthTxEngineIntegrationTest struct {
	StartBlock       uint64
	EndBlock         uint64
	GetBlockByNumber func(uint64) *BlockWithStateRoot
	VMFactory        *EthTrxEngineFactory
}

func (this *EthTxEngineIntegrationTest) Run() {
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
		stateTransitionRequest := &trx_engine.StateTransitionRequest{Block: block.Block}
		if prevBlock != nil {
			stateTransitionRequest.BaseStateRoot = prevBlock.StateRoot
		}
		result, err := ethereumVM.TransitionState(stateTransitionRequest)
		util.PanicIfNotNil(err)
		util.Assert(result.StateRoot == block.StateRoot, result.StateRoot.Hex(), " != ", block.StateRoot.Hex())
		ethereumVM.CommitToDisk()
		prevBlock = block
	}
}

func Test_integration(t *testing.T) {
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
	factory := new(EthTrxEngineFactory)
	//factory.ReadDBConfig = &trx_engine_base.StateDBConfig{
	//	DBFactory: &rocksdb.Factory{
	//		File:                   "/Volumes/A/eth-mainnet/eth_mainnet_rocksdb/state",
	//		ReadOnly:               true,
	//		Parallelism:            concurrent.CPU_COUNT,
	//		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
	//		OptimizeForPointLookup: 1 * 1024,
	//		UseDirectReads:         true,
	//	},
	//}
	//factory.WriteDBConfig = &trx_engine_base.StateDBConfig{DBFactory: new(memory.Factory)}
	//factory.ReadDBConfig = &trx_engine_base.StateDBConfig{DBFactory: new(memory.Factory)}
	factory.ReadDBConfig = &trx_engine_base.StateDBConfig{DBFactory: &rocksdb.Factory{
		//File:                   os.TempDir() + string(os.PathSeparator) + "ololololo",
		File:                   os.TempDir() + string(os.PathSeparator) + "ololololo1",
		Parallelism:            concurrent.CPU_COUNT,
		MaxFileOpeningThreads:  concurrent.CPU_COUNT,
		OptimizeForPointLookup: 3 * 1024,
		MaxOpenFiles:           8192,
	}}
	factory.BlockHashSourceFactory = trx_engine_base.SimpleBlockHashSourceFactory(func(blockNumber uint64) common.Hash {
		return getBlockByNumber(blockNumber).Hash
	})
	test := EthTxEngineIntegrationTest{
		//StartBlock: 1665936,


		//StartBlock: 50220,
		StartBlock: 2691830,
		//StartBlock: 51921,
		//StartBlock: 50222,
		//StartBlock:       477942,
		//StartBlock:       626668,
		EndBlock: 400050223,
		//EndBlock:         4000000,
		GetBlockByNumber: getBlockByNumber,
		VMFactory:        factory,
	}
	test.Run()
}

//panic: 0x33a60cb9131ed0a5e21f4230ca05abda1617d03ef86c0f5595f9f12e35b0947c,  != , 0x371f0defb344a06fa8bf428ce5dac951969a2207ca9ed707642b693305955691 [recovered]
//trx 0 9af6fab899ff4faffa451539faf3a7e26201284c0e40b5941f1c2673036c5441
//trx 1 4683306f2dfb871642b16d89d8fe20ddfdb383ce96fa2c70ea8062a8b2ac1c40
//trx 2 197d16e10543dc3bbf611a6d2892b3d5384629118c4d5f525a12eb34730ac854
//trx 3 5368a59f878f08276235321e0d3d9196133d909ff8630985015d38d0b2cdc6d5
