package vm

import (
	"encoding/json"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/taraxa/db"
	"github.com/Taraxa-project/taraxa-evm/taraxa/metric_utils"
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

type Transaction = struct {
	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    hexutil.Uint64  `json:"nonce"`
	Amount   *hexutil.Big    `json:"amount"`
	GasLimit hexutil.Uint64  `json:"gasLimit"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Data     hexutil.Bytes   `json:"data"`
	Hash     common.Hash     `json:"hash"`
}

type BlockNumberAndCoinbase = struct {
	Number *big.Int `json:"number"`
	// TODO
	//Number   *hexutil.Big   `json:"number"`
	Coinbase common.Address `json:"coinbase"`
}

type BlockHeader = struct {
	BlockNumberAndCoinbase
	Time       *hexutil.Big   `json:"time"`
	Difficulty *hexutil.Big   `json:"difficulty"`
	GasLimit   hexutil.Uint64 `json:"gasLimit"`
	Hash       common.Hash    `json:"hash"`
}

type Block = struct {
	BlockHeader
	UncleBlocks  []*BlockNumberAndCoinbase `json:"uncleBlocks"`
	Transactions []*Transaction            `json:"transactions"`
}

type StateTransitionRequest = struct {
	BaseStateRoot common.Hash `json:"stateRoot"`
	Block         *Block      `json:"block"`
}

type ConcurrentSchedule = struct {
	SequentialTransactions *TxIdSet `json:"sequential"`
}

type TaraxaReceipt = struct {
	ReturnValue     hexutil.Bytes  `json:"returnValue"`
	EthereumReceipt *types.Receipt `json:"ethereumReceipt"`
	ContractError   error          `json:"contractError"`
}

type StateTransitionReceipt = struct {
	Receipts        []*TaraxaReceipt                `json:"receipts"`
	Preimages       map[common.Hash]hexutil.Bytes   `json:"preimages"`
	ChangedBalances map[common.Address]*hexutil.Big `json:"changedBalances"`
	AllLogs         []*types.Log                    `json:"allLogs"`
	UsedGas         hexutil.Uint64                  `json:"usedGas"`
}

type StateTransitionResult = struct {
	StateRoot common.Hash `json:"stateRoot"`
	StateTransitionReceipt
}

type TransactionMetrics = struct {
	TotalTime       metric_utils.AtomicCounter `json:"totalTime"`
	TrieReads       metric_utils.AtomicCounter `json:"trieReads"`
	PersistentReads metric_utils.AtomicCounter `json:"persistentReads"`
}

type StateDBConfig struct {
	DB        *db.GenericFactory `json:"db"`
	CacheSize int                `json:"cacheSize"`
}
