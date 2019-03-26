package main

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"math/big"
)

type Transaction struct {
	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    uint64          `json:"nonce"`
	Amount   *big.Int        `json:"amount"`
	GasLimit uint64          `json:"gasLimit"`
	GasPrice *big.Int        `json:"gasPrice"`
	Data     []byte          `json:"data"`
}

type Block struct {
	Coinbase   common.Address `json:"coinbase"`
	Number     *big.Int       `json:"number"`
	Time       *big.Int       `json:"time"`
	Difficulty *big.Int       `json:"difficulty"`
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
