package trx_engine

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"math/big"
)

type TxIndex = int
type Nonce = uint64
type Balance = *big.Int

type Transaction = struct {
	To       *common.Address `json:"to"`
	From     common.Address  `json:"from" gencodec:"required"`
	Nonce    hexutil.Uint64  `json:"nonce" gencodec:"required"`
	Value    *hexutil.Big    `json:"value" gencodec:"required"`
	Gas      hexutil.Uint64  `json:"gas" gencodec:"required"`
	GasPrice *hexutil.Big    `json:"gasPrice" gencodec:"required"`
	Input    hexutil.Bytes   `json:"input" gencodec:"required"`
	Hash     common.Hash     `json:"hash" gencodec:"required"`
}

type BlockNumberAndCoinbase = struct {
	Number *big.Int `json:"number" gencodec:"required"`
	// TODO blk num hex
	//Number   *hexutil.Big   `json:"number"`
	Miner common.Address `json:"miner" gencodec:"required"`
}

// TODO remove
type UncleBlock = struct {
	Number *hexutil.Big   `json:"number"  gencodec:"required"`
	Miner  common.Address `json:"miner"  gencodec:"required"`
}

type BlockHeader = struct {
	BlockNumberAndCoinbase
	Time       *hexutil.Big   `json:"timestamp"  gencodec:"required"`
	Difficulty *hexutil.Big   `json:"difficulty"  gencodec:"required"`
	GasLimit   hexutil.Uint64 `json:"gasLimit"  gencodec:"required"`
	Hash       common.Hash    `json:"hash"`
}

type Block = struct {
	BlockHeader
	UncleBlocks  []*UncleBlock  `json:"uncleBlocks"  gencodec:"required"`
	Transactions []*Transaction `json:"transactions"  gencodec:"required"`
}

type StateTransitionRequest = struct {
	BaseStateRoot common.Hash `json:"stateRoot"`
	Block         *Block      `json:"block"`
}

type ConcurrentSchedule = struct {
	SequentialTransactions []TxIndex `json:"sequential"`
}

type TransactionOutput = struct {
	ReturnValue hexutil.Bytes `json:"returnValue"`
	// TODO: error codes + messages
	Error error `json:"error"`
}

type StateTransitionResult = struct {
	StateRoot          common.Hash          `json:"stateRoot"`
	Receipts           types.Receipts       `json:"receipts"`
	TransactionOutputs []*TransactionOutput `json:"transactionOutputs"`
	UsedGas            hexutil.Uint64       `json:"usedGas"`
}
