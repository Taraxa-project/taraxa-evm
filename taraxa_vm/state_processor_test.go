package taraxa_vm

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/compiler"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/crypto"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa_vm/conflict_tracking"
	"github.com/stretchr/testify/assert"
	"math/big"
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

	db := ethdb.NewMemDatabase()

	blockData := BlockData{
		Number:     big.NewInt(0),
		Time:       big.NewInt(0),
		Difficulty: big.NewInt(0),
		Coinbase:   *addr(100),
		GasLimit:   100000000000,
	}

	contractCreatingTx1 := TransactionData{
		Nonce:    0,
		From:     *addr(100),
		Data:     code(SingleVariable),
		Amount:   big.NewInt(0),
		GasPrice: big.NewInt(0),
		GasLimit: 100000000,
	}
	contractAddr1 := crypto.CreateAddress(contractCreatingTx1.From, contractCreatingTx1.Nonce)

	contractCreatingTx2 := TransactionData{
		Nonce:    0,
		From:     *addr(101),
		Data:     code(SingleVariable),
		Amount:   big.NewInt(0),
		GasPrice: big.NewInt(0),
		GasLimit: 100000000,
	}
	contractAddr2 := crypto.CreateAddress(contractCreatingTx2.From, contractCreatingTx2.Nonce)

	result1, err := Process(db, &StateTransition{
		StateRoot: common.Hash{},
		BlockData: &blockData,
		Transactions: []*TransactionData{
			&contractCreatingTx1,
			&contractCreatingTx2,
		},
	}, nil)
	failOn(err)

	conflicts1 := result1.Conflicts.GetConflictingTransactions()
	assert.True(t, len(conflicts1) == 0)

	result2, err := Process(db, &StateTransition{
		StateRoot: result1.StateRoot,
		BlockData: &blockData,
		Transactions: []*TransactionData{
			&TransactionData{
				Nonce:    0,
				From:     *addr(102),
				To:       &contractAddr1,
				Data:     call(SingleVariable, "set", big.NewInt(2)),
				Amount:   big.NewInt(0),
				GasPrice: big.NewInt(0),
				GasLimit: 100000000,
			},
			&TransactionData{
				Nonce:    0,
				From:     *addr(103),
				To:       &contractAddr2,
				Data:     call(SingleVariable, "set", big.NewInt(3)),
				Amount:   big.NewInt(0),
				GasPrice: big.NewInt(0),
				GasLimit: 100000000,
			},
			&TransactionData{
				Nonce:    1,
				From:     *addr(103),
				To:       &contractAddr2,
				Data:     call(SingleVariable, "get"),
				Amount:   big.NewInt(0),
				GasPrice: big.NewInt(0),
				GasLimit: 100000000,
			},
		},
	}, nil)
	failOn(err)

	conflicts2 := result2.Conflicts.GetConflictingTransactions()
	assert.Equal(t, conflicts2, []conflict_tracking.TxId{1, 2})
}

func addr(n int64) *common.Address {
	ret := new(common.Address)
	*ret = common.BigToAddress(big.NewInt(n))
	return ret
}

func compile(contract string) *compiler.Contract {
	contracts, err := compiler.CompileSolidityString("solc", contract);
	failOn(err)
	for _, contract := range contracts {
		return contract
	}
	panic("no contracts in the output")
}

func code(contract *compiler.Contract) []byte {
	code, err := hexutil.Decode(contract.Code)
	failOn(err)
	return code
}

func call(contract *compiler.Contract, method string, args ...interface{}) []byte {
	calldata, err := contract.Info.AbiDefinition.Pack(method, args...)
	failOn(err)
	return calldata;
}

func failOn(err error) {
	if err != nil {
		panic(err);
	}
}

func blackHole(...interface{}) {

}
