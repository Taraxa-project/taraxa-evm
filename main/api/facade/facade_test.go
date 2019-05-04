package facade

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

	stateDBConfig, stateDBCleanup := newTestLDB("state")
	blockchainDBConfig, blockchainDBCleanup := newTestLDB("blockchain")
	defer stateDBCleanup()
	defer blockchainDBCleanup()

	requestProto := api.Request{
		StateDatabase:     &stateDBConfig,
		BlockHashDatabase: &blockchainDBConfig,
		StateTransition: &api.StateTransition{
			Block: &api.Block{
				Number:     "0",
				Time:       "0",
				Difficulty: "0",
				Coinbase:   *addr(99),
				GasLimit:   100000000000,
			},
		},
	}

	request1 := api.Request(requestProto)
	request1.StateTransition.Transactions = []*api.Transaction{
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
	}
	scheduleResponse1 := Run(&request1)
	util.PanicIfPresent(scheduleResponse1.Error)

	util.Assert(len(scheduleResponse1.ConcurrentSchedule.Sequential) == 0)

	request1.ConcurrentSchedule = scheduleResponse1.ConcurrentSchedule
	stateTransitionResponse1 := Run(&request1)
	util.PanicIfPresent(stateTransitionResponse1.Error)

	receipts1 := stateTransitionResponse1.StateTransitionResult.Receipts
	contractAddr1 := receipts1[0].EthereumReceipt.ContractAddress
	contractAddr2 := receipts1[1].EthereumReceipt.ContractAddress
	someValue := big.NewInt(66)

	request2 := api.Request(requestProto)
	request2.StateTransition.StateRoot = stateTransitionResponse1.StateTransitionResult.StateRoot
	request2.StateTransition.Transactions = []*api.Transaction{
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
	}

	scheduleResponse2 := Run(&request2)
	util.PanicIfPresent(scheduleResponse2.Error)

	AssertContainsExactly(t, scheduleResponse2.ConcurrentSchedule.Sequential, 1, 2)

	request2.ConcurrentSchedule = scheduleResponse2.ConcurrentSchedule
	stateTransitionResponse2 := Run(&request2)
	util.PanicIfPresent(stateTransitionResponse2.Error)

	receipts2 := stateTransitionResponse2.StateTransitionResult.Receipts
	assert.Equal(t, hexutil.Bytes(common.BigToHash(someValue).Bytes()), receipts2[2].ReturnValue)

	request3 := api.Request(requestProto)
	request3.StateTransition.StateRoot = stateTransitionResponse2.StateTransitionResult.StateRoot
	request3.StateTransition.Transactions = []*api.Transaction{
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
	}
	scheduleResponse3 := Run(&request3)
	util.PanicIfPresent(scheduleResponse3.Error)

	AssertContainsExactly(t, scheduleResponse3.ConcurrentSchedule.Sequential, 0, 1)

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
	util.PanicIfPresent(err)
	for _, contract := range contracts {
		return contract
	}
	panic("no contracts in the output")
}

func code(contract *compiler.Contract) *hexutil.Bytes {
	var code hexutil.Bytes
	code, err := hexutil.Decode(contract.Code)
	util.PanicIfPresent(err)
	return &code
}

func call(contract *compiler.Contract, method string, args ...interface{}) *hexutil.Bytes {
	var calldata hexutil.Bytes
	calldata, err := contract.Info.AbiDefinition.Pack(method, args...)
	util.PanicIfPresent(err)
	return &calldata;
}

func newTestLDB(name string) (api.LDBConfig, func()) {
	dirname := "__test_ldb__" + name
	if _, err := os.Stat(dirname); !os.IsNotExist(err) {
		util.PanicIfPresent(os.RemoveAll(dirname))
	}
	util.PanicIfPresent(os.Mkdir(dirname, os.ModePerm))
	absPath, err := filepath.Abs(dirname)
	util.PanicIfPresent(err)
	return api.LDBConfig{
		File:    absPath,
		Cache:   0,
		Handles: 0,
	}, func() {
		util.PanicIfPresent(os.RemoveAll(dirname))
	}
}

func blackHole(...interface{}) {

}
