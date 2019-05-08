package api

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/main/metrics"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"math/big"
)

type TxId = int

type BigIntString = *big.Int

type BlockHashStore interface {
	GetHeaderHashByBlockNumber(blockNumber uint64) common.Hash
}

type ExternalApi interface {
	BlockHashStore
}

type StateDBConfig struct {
	DbConfig  *GenericDbConfig `json:"db"`
	CacheSize int              `json:"cacheSize"`
}

type Transaction struct {
	To       *common.Address `json:"to"`
	From     common.Address  `json:"from"`
	Nonce    uint64          `json:"nonce"`
	Amount   BigIntString    `json:"amount"`
	GasLimit uint64          `json:"gasLimit"`
	GasPrice BigIntString    `json:"gasPrice"`
	Data     hexutil.Bytes   `json:"data"`
	Hash     common.Hash     `json:"hash"`
}

func (this *Transaction) AsMessage(checkNonce bool) types.Message {
	return types.NewMessage(
		this.From, this.To, this.Nonce, this.Amount, this.GasLimit, this.GasPrice, this.Data,
		checkNonce,
	)
}

type HeaderNumerAndCoinbase struct {
	Number   BigIntString   `json:"number"`
	Coinbase common.Address `json:"coinbase"`
}

type BlockHeader struct {
	HeaderNumerAndCoinbase
	Time       BigIntString `json:"time"`
	Difficulty BigIntString `json:"difficulty"`
	GasLimit   uint64       `json:"gasLimit"`
	Hash       common.Hash  `json:"hash"`
}

type Block struct {
	BlockHeader
	Uncles       []*HeaderNumerAndCoinbase `json:"uncles"`
	Transactions []*Transaction            `json:"transactions"`
}

type StateTransitionRequest struct {
	BaseStateRoot common.Hash `json:"stateRoot"`
	ExpectedRoot  common.Hash `json:"expectedRoot"`
	Block         *Block      `json:"block"`
}

type ConcurrentSchedule struct {
	SequentialTransactions util.LinkedHashSet `json:"sequential"`
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

type StateTransitionResponse struct {
	Result StateTransitionResult `json:"result"`
	Error  *util.SimpleError     `json:"error"`
}

type VMConfig struct {
	StateDB                 StateDBConfig    `json:"stateDB"`
	Evm                     *vm.StaticConfig `json:"evm"`
	Genesis                 *core.Genesis    `json:"genesis"`
	BlockHashDB             *GenericDbConfig `json:"blockDB"`
	StateTransitionTargetDB *GenericDbConfig `json:"stateTransitionTargetDB"`
}

type TransactionMetrics struct {
	TotalExecutionTime metrics.AtomicCounter
	TotalTrieReadTime  metrics.AtomicCounter
}

type Metrics struct {
	TransactionMetrics []TransactionMetrics
}
