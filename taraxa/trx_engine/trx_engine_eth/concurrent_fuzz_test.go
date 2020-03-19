package trx_engine_eth

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/state"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/db/memory"
	"github.com/Taraxa-project/taraxa-evm/taraxa/trx_engine/trx_engine_base"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/concurrent"
	"math/big"
	"testing"
)

// use -race go tool flag
func Test_concurrent_fuzz(t *testing.T) {
	var addrs [10]common.Address
	for i := 0; i < len(addrs); i++ {
		addrs[i] = common.BigToAddress(big.NewInt(int64(100 + i)))
	}
	var engines [5]*EthTrxEngine
	var base_root *common.Hash
	for i := 0; i < len(engines); i++ {
		var factory EthTrxEngineFactory
		factory.Genesis = trx_engine.TaraxaGenesisConfig
		factory.DisableMinerReward = true
		factory.DisableNonceCheck = true
		factory.DisableGasFee = true
		factory.DBConfig = &trx_engine_base.StateDBConfig{
			DBFactory: &memory.Factory{},
		}
		factory.BlockHashSourceFactory = trx_engine_base.SimpleBlockHashSourceFactory(func(uint64) (ret common.Hash) {
			panic("block hash by number is not implemented")
		})
		engine, _, err_0 := factory.NewInstance()
		util.PanicIfNotNil(err_0)
		state_db := state.New(common.Hash{}, engine.DB)
		for i := 0; i < len(addrs); i++ {
			state_db.SetBalance(addrs[i], big.NewInt(1000))
		}
		state_db.Checkpoint(true)
		root := state_db.Commit()
		engine.DB.Commit()
		if base_root == nil {
			base_root = &root
		} else {
			util.Assert(*base_root == root)
		}
		engines[i] = engine
	}
	done := concurrent.NewRendezvous(len(engines))
	for i := 0; i < len(engines); i++ {
		i := i
		go func() {
			defer done.CheckIn()
			engine := engines[i]
			var req trx_engine.StateTransitionRequest
			req.Block = new(trx_engine.Block)
			req.Block.Number = big.NewInt(1)
			req.Block.Transactions = []*trx_engine.Transaction{
				&trx_engine.Transaction{
					From:  addrs[0],
					To:    &addrs[1],
					Value: (*hexutil.Big)(big.NewInt(1)),
				},
			}
			engine.TransitionStateAndCommit(&req)
		}()
	}
	done.Await()
}
