package api

import (
	"errors"
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"
	"github.com/Taraxa-project/taraxa-evm/core/vm"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/main/util"
	"github.com/Taraxa-project/taraxa-evm/params"
	"math/big"
)

type TxId = int
type BigIntString = string;

type BlockHashStore interface {
	GetHeaderHashByBlockNumber(blockNumber uint64) common.Hash
}

type ExternalApi interface {
	BlockHashStore
}

func BigInt(str BigIntString) *big.Int {
	if ret, success := new(big.Int).SetString(str, 10); success {
		return ret
	}
	panic(errors.New("Could not convert string to bigint: " + str))
}

type LDBConfig struct {
	File    string `json:"file"`
	Cache   int    `json:"cache"`
	Handles int    `json:"handles"`
}

func (this *LDBConfig) NewLdbDatabase() *ethdb.LDBDatabase {
	db, err := ethdb.NewLDBDatabase(this.File, this.Cache, this.Handles)
	util.PanicIfPresent(err)
	return db
}

type StateDBConfig struct {
	LDB       *LDBConfig `json:"ldb"`
	CacheSize int        `json:"cacheSize"`
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
	Coinbase     common.Address `json:"coinbase"`
	Number       BigIntString   `json:"number"`
	Time         BigIntString   `json:"time"`
	Difficulty   BigIntString   `json:"difficulty"`
	GasLimit     uint64         `json:"gasLimit"`
	Hash         common.Hash    `json:"hash"`
	Uncles       []common.Hash  `json:"uncles"`
	Transactions []*Transaction `json:"transactions"`
}

type StateTransition struct {
	StateRoot common.Hash `json:"stateRoot"`
	Block     *Block      `json:"block"`
}

type ScheduleRequest struct {
	StateTransition *StateTransition `json:"stateTransition" gencodec:"required"`
}

type ConcurrentSchedule struct {
	Sequential []TxId `json:"sequential"`
}

type ScheduleResponse struct {
	Result *ConcurrentSchedule `json:"result"`
	Error  *util.SimpleError   `json:"error"`
}

type StateTransitionRequest struct {
	StateTransition    *StateTransition    `json:"stateTransition" gencodec:"required"`
	ConcurrentSchedule *ConcurrentSchedule `json:"concurrentSchedule"`
	TargetLevelDB      *LDBConfig          `json:"targetLevelDB"`
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
	StateDB                  StateDBConfig       `json:"stateDB"`
	Evm                      *vm.StaticConfig    `json:"evm"`
	Chain                    *params.ChainConfig `json:"chain"`
	BlockHashLDB             *LDBConfig          `json:"blockHashLDB"`
	StateTransitionTargetLDB *LDBConfig          `json:"stateTransitionTargetLDB"`
}
