package state_transition

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/compiler"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/main/api"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"path/filepath"
	"testing"
)

func TestConflictDetection(t *testing.T) {
	SingleVariable := compile(`
pragma solidity ^0.5.6;

contract SingleVariable {

	mapping(uint => uint) m;

	function set(uint key, uint value) public {
		m[key] = value;
    }

    function get(uint key) public view returns (uint) {
        return m[key];
    }

}
	`)

	ldbConfig, ldbCleanup := newTestLDB()
	defer ldbCleanup()

	externalApi := new(api.ExternalApi)

	block := api.Block{
		Number:     "0",
		Time:       "0",
		Difficulty: "0",
		Coinbase:   *addr(99),
		GasLimit:   100000000000,
	}

	result1, err := Run(&api.RunConfiguration{
		LDBConfig: &ldbConfig,
		StateTransition: api.StateTransition{
			StateRoot: common.Hash{},
			Block:     &block,
			Transactions: []*api.Transaction{
				&api.Transaction{
					Nonce:    0,
					From:     *addr(100),
					Data:     code(SingleVariable),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
				&api.Transaction{
					Nonce:    0,
					From:     *addr(101),
					Data:     code(SingleVariable),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
			},
		},
	}, externalApi)
	util.PanicOn(err)

	contractAddr1 := result1.Receipts[0].EthereumReceipt.ContractAddress
	contractAddr2 := result1.Receipts[1].EthereumReceipt.ContractAddress

	util.Assert(len(result1.ConcurrentSchedule.Sequential) == 0)

	someValue := big.NewInt(66)
	result2, err := Run(&api.RunConfiguration{
		LDBConfig: &ldbConfig,
		StateTransition: api.StateTransition{
			StateRoot: result1.StateRoot,
			Block:     &block,
			Transactions: []*api.Transaction{
				&api.Transaction{
					Nonce:    0,
					From:     *addr(102),
					To:       &contractAddr1,
					Data:     call(SingleVariable, "set", big.NewInt(0), big.NewInt(4)),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
				&api.Transaction{
					Nonce:    0,
					From:     *addr(103),
					To:       &contractAddr2,
					Data:     call(SingleVariable, "set", big.NewInt(3), someValue),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
				&api.Transaction{
					Nonce:    0,
					From:     *addr(104),
					To:       &contractAddr2,
					Data:     call(SingleVariable, "get", big.NewInt(3)),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
			},
		},
	}, externalApi)
	util.PanicOn(err)

	AssertContainsExactly(t, result2.ConcurrentSchedule.Sequential, 1, 2)
	assert.Equal(t, hexutil.Bytes(common.BigToHash(someValue).Bytes()), result2.Receipts[2].ReturnValue)

	result3, err := Run(&api.RunConfiguration{
		LDBConfig: &ldbConfig,
		StateTransition: api.StateTransition{
			StateRoot: result1.StateRoot,
			Block:     &block,
			Transactions: []*api.Transaction{
				&api.Transaction{
					Nonce:    0,
					From:     *addr(102),
					To:       &contractAddr1,
					Data:     call(SingleVariable, "set", big.NewInt(5), big.NewInt(0)),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
				&api.Transaction{
					Nonce:    0,
					From:     *addr(103),
					To:       &contractAddr1,
					Data:     call(SingleVariable, "set", big.NewInt(5), big.NewInt(1)),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
				&api.Transaction{
					Nonce:    0,
					From:     *addr(104),
					To:       &contractAddr2,
					Data:     call(SingleVariable, "get", big.NewInt(0)),
					Amount:   "0",
					GasPrice: "0",
					GasLimit: 100000000,
				},
			},
		},
	}, externalApi)
	util.PanicOn(err)

	AssertContainsExactly(t, result3.ConcurrentSchedule.Sequential, 0, 1)

}

func AssertContainsExactly(t *testing.T, left []api.TxId, right ...api.TxId) {
	assert.Equal(t, set(right...), set(left...))
}

func set(ids ...api.TxId) []interface{} {
	set := treeset.NewWithIntComparator()
	for _, id := range ids {
		set.Add(id)
	}
	return set.Values()
}

func addr(n int64) *common.Address {
	ret := new(common.Address)
	*ret = common.BigToAddress(big.NewInt(n))
	return ret
}

func compile(contract string) *compiler.Contract {
	contracts, err := compiler.CompileSolidityString("solc", contract);
	util.PanicOn(err)
	for _, contract := range contracts {
		return contract
	}
	panic("no contracts in the output")
}

func code(contract *compiler.Contract) *hexutil.Bytes {
	var code hexutil.Bytes
	code, err := hexutil.Decode(contract.Code)
	util.PanicOn(err)
	return &code
}

func call(contract *compiler.Contract, method string, args ...interface{}) *hexutil.Bytes {
	var calldata hexutil.Bytes
	calldata, err := contract.Info.AbiDefinition.Pack(method, args...)
	util.PanicOn(err)
	return &calldata;
}

func newTestLDB() (api.LDBConfig, func()) {
	dirname := "__test_ldb__"
	if _, err := os.Stat(dirname); !os.IsNotExist(err) {
		util.PanicOn(os.RemoveAll(dirname))
	}
	util.PanicOn(os.Mkdir(dirname, os.ModePerm))
	absPath, err := filepath.Abs(dirname)
	util.PanicOn(err)
	return api.LDBConfig{
		File:    absPath,
		Cache:   0,
		Handles: 0,
	}, func() {
		util.PanicOn(os.RemoveAll(dirname))
	}
}

func blackHole(...interface{}) {

}
