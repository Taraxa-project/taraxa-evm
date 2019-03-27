package main

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"math/big"
)

type BigIntString = string;

func BigInt(str BigIntString) *big.Int {
	if ret, success := new(big.Int).SetString(str, 10); success {
		return ret
	}
	panic(errors.New("Could not convert string to bigint: " + str))
}

type Transaction struct {
	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    uint64          `json:"nonce"`
	Amount   BigIntString    `json:"amount"`
	GasLimit uint64          `json:"gasLimit"`
	GasPrice BigIntString    `json:"gasPrice"`
	Data     []byte          `json:"data"`
}

type Block struct {
	Coinbase   common.Address `json:"coinbase"`
	Number     BigIntString   `json:"number"`
	Time       BigIntString   `json:"time"`
	Difficulty BigIntString   `json:"difficulty"`
	GasLimit   uint64         `json:"gasLimit"`
	Hash       common.Hash    `json:"hash"`
}

type ConcurrentSchedule struct {
	Sequential []uint64 `json:"sequential"`
}

type LDBConfig struct {
	File    string `json:"file"`
	Cache   int    `json:"cache"`
	Handles int    `json:"handles"`
}

type RunConfiguration struct {
	StateRoot          common.Hash         `json:"stateRoot"`
	Block              *Block              `json:"block"`
	Transactions       []*Transaction      `json:"transactions"`
	LDBConfig          *LDBConfig          `json:"ldbConfig"`
	ConcurrentSchedule *ConcurrentSchedule `json:"concurrentSchedule"`
}

type Result struct {
	StateRoot          common.Hash         `json:"stateRoot"`
	ConcurrentSchedule *ConcurrentSchedule `json:"concurrentSchedule"`
	Receipts           types.Receipts      `json:"receipts"`
	AllLogs            []*types.Log        `json:"allLogs"`
	UsedGas            uint64              `json:"usedGas"`
	ReturnValues       [][]byte            `json:"returnValues"`
	Error              error               `json:"error"`
}
