package api

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

type TxId = int
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
	Data     *hexutil.Bytes  `json:"data"`
	Hash     common.Hash     `json:"hash"`
}

type Block struct {
	Coinbase   common.Address `json:"coinbase"`
	Number     BigIntString   `json:"number"`
	Time       BigIntString   `json:"time"`
	Difficulty BigIntString   `json:"difficulty"`
	GasLimit   uint64         `json:"gasLimit"`
	Hash       common.Hash    `json:"hash"`
}

type StateTransition struct {
	StateRoot    common.Hash    `json:"stateRoot"`
	Block        *Block         `json:"block"`
	Transactions []*Transaction `json:"transactions"`
}

type ConcurrentSchedule struct {
	Sequential []TxId `json:"sequential"`
}

type LDBConfig struct {
	File    string `json:"file"`
	Cache   int    `json:"cache"`
	Handles int    `json:"handles"`
}

type TaraxaReceipt struct {
	ReturnValue     hexutil.Bytes  `json:"returnValue"`
	EthereumReceipt *types.Receipt `json:"ethereumReceipt"`
	ContractError   error          `json:"contractError"`
}

type StateTransitionResult struct {
	StateRoot common.Hash      `json:"stateRoot"`
	Receipts  []*TaraxaReceipt `json:"receipts"`
	AllLogs   []*types.Log     `json:"allLogs"`
	UsedGas   uint64           `json:"usedGas"`
}

type Request struct {
	StateTransition    *StateTransition    `json:"stateTransition"`
	StateDatabase      *LDBConfig          `json:"stateDatabase"`
	BlockchainDatabase *LDBConfig          `json:"blockchainDatabase"`
	ConcurrentSchedule *ConcurrentSchedule `json:"concurrentSchedule"`
}

type Response struct {
	StateTransitionResult *StateTransitionResult `json:"stateTransitionResult"`
	ConcurrentSchedule    *ConcurrentSchedule    `json:"concurrentSchedule"`
	Error                 *util.SimpleError      `json:"error"`
}
