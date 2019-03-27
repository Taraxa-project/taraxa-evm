package main

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/compiler"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/main/conflict_tracking"
	"github.com/Taraxa-project/taraxa-evm/main/util"
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

    uint value = 5;

	function set(uint _value) public {
		value = _value;
    }

    function get() public view returns (uint) {
        return value;
    }

}
	`)

	ldbConfig, ldbCleanup := newTestLDB()
	defer ldbCleanup()

	block := Block{
		Number:     "0",
		Time:       "0",
		Difficulty: "0",
		Coinbase:   *addr(100),
		GasLimit:   100000000000,
	}

	contractCreatingTx1 := Transaction{
		Nonce:    0,
		From:     *addr(100),
		Data:     code(SingleVariable),
		Amount:   "0",
		GasPrice: "0",
		GasLimit: 100000000,
	}
	contractAddr1 := crypto.CreateAddress(contractCreatingTx1.From, contractCreatingTx1.Nonce)

	contractCreatingTx2 := Transaction{
		Nonce:    0,
		From:     *addr(101),
		Data:     code(SingleVariable),
		Amount:   "0",
		GasPrice: "0",
		GasLimit: 100000000,
	}
	contractAddr2 := crypto.CreateAddress(contractCreatingTx2.From, contractCreatingTx2.Nonce)

	result1, err := Process(&RunConfiguration{
		StateRoot: common.Hash{},
		Block:     &block,
		LDBConfig: &ldbConfig,
		Transactions: []*Transaction{
			&contractCreatingTx1,
			&contractCreatingTx2,
		},
	})
	util.FailOnErr(err)

	assert.True(t, len(result1.ConcurrentSchedule.Sequential) == 0)

	result2, err := Process(&RunConfiguration{
		StateRoot: result1.StateRoot,
		Block:     &block,
		LDBConfig: &ldbConfig,
		Transactions: []*Transaction{
			&Transaction{
				Nonce:    0,
				From:     *addr(102),
				To:       &contractAddr1,
				Data:     call(SingleVariable, "set", big.NewInt(2)),
				Amount:   "0",
				GasPrice: "0",
				GasLimit: 100000000,
			},
			&Transaction{
				Nonce:    0,
				From:     *addr(103),
				To:       &contractAddr2,
				Data:     call(SingleVariable, "set", big.NewInt(3)),
				Amount:   "0",
				GasPrice: "0",
				GasLimit: 100000000,
			},
			&Transaction{
				Nonce:    1,
				From:     *addr(103),
				To:       &contractAddr2,
				Data:     call(SingleVariable, "get"),
				Amount:   "0",
				GasPrice: "0",
				GasLimit: 100000000,
			},
		},
	})
	util.FailOnErr(err)

	assert.Equal(t, result2.ConcurrentSchedule.Sequential, []conflict_tracking.TxId{1, 2})
}

func addr(n int64) *common.Address {
	ret := new(common.Address)
	*ret = common.BigToAddress(big.NewInt(n))
	return ret
}

func compile(contract string) *compiler.Contract {
	contracts, err := compiler.CompileSolidityString("solc", contract);
	util.FailOnErr(err)
	for _, contract := range contracts {
		return contract
	}
	panic("no contracts in the output")
}

func code(contract *compiler.Contract) *hexutil.Bytes {
	var code hexutil.Bytes
	code, err := hexutil.Decode(contract.Code)
	util.FailOnErr(err)
	return &code
}

func call(contract *compiler.Contract, method string, args ...interface{}) *hexutil.Bytes {
	var calldata hexutil.Bytes
	calldata, err := contract.Info.AbiDefinition.Pack(method, args...)
	util.FailOnErr(err)
	return &calldata;
}

func newTestLDB() (LDBConfig, func()) {
	dirname := "__test_ldb__"
	if _, err := os.Stat(dirname); !os.IsNotExist(err) {
		util.FailOnErr(os.RemoveAll(dirname))
	}
	util.FailOnErr(os.Mkdir(dirname, os.ModePerm))
	absPath, err := filepath.Abs(dirname)
	util.FailOnErr(err)
	return LDBConfig{
		File:    absPath,
		Cache:   0,
		Handles: 0,
	}, func() {
		util.FailOnErr(os.RemoveAll(dirname))
	}
}

func blackHole(...interface{}) {

}
