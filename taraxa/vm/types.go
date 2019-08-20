package vm

import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"math/big"
)

type TxId = int

type TxIdSet struct {
	*util.LinkedHashSet
}

func NewTxIdSet(arr interface{}) *TxIdSet {
	return &TxIdSet{util.NewLinkedHashSet(arr)}
}

func (this *TxIdSet) UnmarshalJSON(data []byte) error {
	elements := []TxId{}
	err := json.Unmarshal(data, &elements)
	if err == nil {
		this.LinkedHashSet = util.NewLinkedHashSet(elements)
	}
	return err
}

type Transaction struct {
	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    uint64          `json:"nonce"`
	Amount   *big.Int        `json:"amount"`
	GasLimit uint64          `json:"gasLimit"`
	GasPrice *big.Int        `json:"gasPrice"`
	Data     hexutil.Bytes   `json:"data"`
	Hash     common.Hash     `json:"hash"`
}

type BlockNumberAndCoinbase struct {
	Number   *big.Int       `json:"number"`
	Coinbase common.Address `json:"coinbase"`
}

type BlockHeader struct {
	BlockNumberAndCoinbase
	Time       *big.Int    `json:"time"`
	Difficulty *big.Int    `json:"difficulty"`
	GasLimit   uint64      `json:"gasLimit"`
	Hash       common.Hash `json:"hash"`
}

type Block struct {
	BlockHeader
	Uncles       []*BlockNumberAndCoinbase `json:"uncles"`
	Transactions []*Transaction            `json:"transactions"`
}

type StateTransitionRequest struct {
	BaseStateRoot common.Hash `json:"stateRoot"`
	Block         *Block      `json:"block"`
}

type ConcurrentSchedule struct {
	SequentialTransactions *TxIdSet `json:"sequential"`
}

type TaraxaReceipt struct {
	ReturnValue     hexutil.Bytes  `json:"returnValue"`
	EthereumReceipt *types.Receipt `json:"ethereumReceipt"`
	ContractError   error          `json:"contractError"`
}

type StateTransitionReceipt struct {
	Receipts []*TaraxaReceipt `json:"receipts"`
	AllLogs  []*types.Log     `json:"allLogs"`
	UsedGas  uint64           `json:"usedGas"`
}

type StateTransitionResult struct {
	StateRoot common.Hash `json:"stateRoot"`
	StateTransitionReceipt
}

type StateTransitionResponse struct {
	Result StateTransitionResult `json:"result"`
	Error  *util.ErrorString     `json:"error"`
}
