package state

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/asserts"

	"github.com/Taraxa-project/taraxa-evm/rlp"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bigutil"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/dpos"

	"github.com/Taraxa-project/taraxa-evm/core/vm"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/tests"

	"github.com/schollz/progressbar/v3"

	"github.com/Taraxa-project/taraxa-evm/taraxa/state/state_db_rocksdb"

	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/params"
	"github.com/Taraxa-project/taraxa-evm/taraxa/data"
)

var simple_chain_cfg = ChainConfig{
	DisableBlockRewards: true,
	ExecutionOptions: vm.ExecutionOpts{
		DisableGasFee:     true,
		DisableNonceCheck: true,
	},
	ETHChainConfig: params.ChainConfig{
		DAOForkBlock: types.BlockNumberNIL,
	},
	GenesisBalances: make(core.BalanceMap),
}

func TestEthMainnetSmoke(t *testing.T) {
	ctx := tests.NewTestCtx(t)
	defer ctx.Close()

	blocks := data.Parse_eth_mainnet_blocks_0_300000()
	statedb := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: ctx.DataDir(),
	})
	defer statedb.Close()
	SUT := new(API).Init(
		statedb,
		func(num types.BlockNum) *big.Int {
			return new(big.Int).SetBytes(blocks[num].Hash[:])
		},
		ChainConfig{
			ETHChainConfig:  *params.MainnetChainConfig,
			GenesisBalances: core.MainnetGenesisBalances(),
		},
		Opts{
			ExpectedMaxTrxPerBlock:        300,
			MainTrieFullNodeLevelsToCache: 4,
		},
	)
	defer SUT.Close()
	st := SUT.GetStateTransition()
	asserts.EQ(statedb.GetLatestState().GetCommittedDescriptor().StateRoot.Hex(), blocks[0].StateRoot.Hex())
	progress_bar := progressbar.Default(int64(len(blocks)))
	defer progress_bar.Finish()
	for blk_num := 1; blk_num < len(blocks); blk_num++ {
		blk := blocks[blk_num]
		st.BeginBlock(&blk.EVMBlock)
		for i := range blk.Transactions {
			st.ExecuteTransaction(&blk.Transactions[i])
		}
		st.EndBlock(blk.UncleBlocks)
		asserts.EQ(st.Commit().Hex(), blk.StateRoot.Hex())
		progress_bar.Add(1)
	}
}

func TestDPOS(t *testing.T) {
	tc := tests.NewTestCtx(t)
	defer tc.Close()

	addr_1, addr_2, addr_3 := tests.SimpleAddr(1), tests.SimpleAddr(2), tests.SimpleAddr(3)
	addr_1_bal_expected := bigutil.Big0
	addr_1_expected_bal_add := func(val *big.Int) *big.Int {
		addr_1_bal_expected = bigutil.Add(addr_1_bal_expected, val)
		return val
	}
	addr_1_expected_bal_sub := func(val *big.Int) *big.Int {
		addr_1_bal_expected = bigutil.USub(addr_1_bal_expected, val)
		return val
	}

	chain_cfg := simple_chain_cfg
	chain_cfg.GenesisBalances[addr_1] = addr_1_expected_bal_add(new(big.Int).SetUint64(100000000))
	dpos_threshold := new(big.Int).SetUint64(1000)
	chain_cfg.DPOS = &dpos.Config{
		DepositDelay:                2,
		WithdrawalDelay:             4,
		EligibilityBalanceThreshold: dpos_threshold,
		GenesisState: dpos.DelegatedBalanceMap{
			addr_1: dpos.BalanceMap{
				addr_1: addr_1_expected_bal_sub(dpos_threshold),
			},
		},
	}

	var curr_blk_n types.BlockNum
	statedb := new(state_db_rocksdb.DB).Init(state_db_rocksdb.Opts{
		Path: tc.DataDir(),
	})
	defer statedb.Close()
	SUT := new(API).Init(
		statedb,
		func(num types.BlockNum) *big.Int {
			panic("unexpected")
		},
		chain_cfg,
		Opts{},
	)

	var expected_eligible_set []common.Address
	CHECK := func() {
		assert_meta := "at block " + fmt.Sprint(curr_blk_n)
		var bal_actual *big.Int
		SUT.ReadBlock(curr_blk_n).GetAccount(&addr_1, func(account state_db.Account) {
			bal_actual = account.Balance
		})
		tc.Assert.Equal(bal_actual, addr_1_bal_expected, assert_meta)
		dpos_reader := SUT.QueryDPOS(curr_blk_n)
		tc.Assert.Equal(dpos_reader.EligibleAddressCount(), uint64(len(expected_eligible_set)), assert_meta)
		for _, addr := range expected_eligible_set {
			tc.Assert.True(dpos_reader.IsEligible(&addr), assert_meta)
		}
	}
	EXEC_AND_CHECK := func(trxs ...vm.Transaction) {
		st := SUT.GetStateTransition()
		st.BeginBlock(new(vm.BlockInfo))
		for i := range trxs {
			res := st.ExecuteTransaction(&trxs[i])
			tc.Assert.Equal(res.ConsensusErr, util.ErrorString(""))
			tc.Assert.Equal(res.CodeErr, util.ErrorString(""))
		}
		st.EndBlock(nil)
		st.Commit()
		curr_blk_n++
		CHECK()
	}

	dpos_transfers := make(dpos.InboundTransfers)
	make_dpos_trx := func() vm.Transaction {
		tmp := dpos_transfers
		dpos_transfers = make(dpos.InboundTransfers)
		dpos_addr := dpos.ContractAddress()
		return vm.Transaction{
			Value: bigutil.Big0,
			To:    &dpos_addr,
			From:  addr_1,
			Input: rlp.MustEncodeToBytes(tmp),
		}
	}

	expected_eligible_set = []common.Address{addr_1}
	CHECK()

	dpos_transfers[addr_2] = dpos.Transfer{Value: addr_1_expected_bal_sub(dpos_threshold)}
	dpos_transfers[addr_3] = dpos.Transfer{Value: addr_1_expected_bal_sub(new(big.Int).Sub(dpos_threshold, bigutil.Big1))}
	EXEC_AND_CHECK(make_dpos_trx())

	withdrawal_val := bigutil.Big1
	dpos_transfers[addr_2] = dpos.Transfer{Value: withdrawal_val, Negative: true}
	dpos_transfers[addr_3] = dpos.Transfer{Value: addr_1_expected_bal_sub(bigutil.Big1)}
	EXEC_AND_CHECK(make_dpos_trx())

	expected_eligible_set = []common.Address{addr_1, addr_2}
	EXEC_AND_CHECK()

	expected_eligible_set = []common.Address{addr_1, addr_2, addr_3}
	EXEC_AND_CHECK()
	EXEC_AND_CHECK()

	addr_1_expected_bal_add(withdrawal_val)
	expected_eligible_set = []common.Address{addr_1, addr_3}
	EXEC_AND_CHECK()
	EXEC_AND_CHECK()
	EXEC_AND_CHECK()
	EXEC_AND_CHECK()
}
