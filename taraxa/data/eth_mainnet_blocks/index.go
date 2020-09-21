package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/common/hexutil"
	"github.com/Taraxa-project/taraxa-evm/core/types"

	"github.com/Taraxa-project/taraxa-evm/taraxa/util/bin"
	"github.com/tecbot/gorocksdb"
)

type Block struct {
	Number       uint64         `json:"number"  gencodec:"required"`
	Miner        common.Address `json:"miner" gencodec:"required"`
	GasLimit     hexutil.Uint64 `json:"gasLimit"  gencodec:"required"`
	Time         hexutil.Uint64 `json:"timestamp"  gencodec:"required"`
	Difficulty   *hexutil.Big   `json:"difficulty"  gencodec:"required"`
	UncleBlocks  []UncleBlock   `json:"uncleBlocks"  gencodec:"required"`
	Transactions []Transaction  `json:"transactions"  gencodec:"required"`
	Hash         common.Hash    `json:"hash" gencodec:"required"`
	StateRoot    common.Hash    `json:"stateRoot" gencodec:"required"`
}
type UncleBlock struct {
	Number hexutil.Uint64 `json:"number"  gencodec:"required"`
	Miner  common.Address `json:"miner"  gencodec:"required"`
}
type Transaction struct {
	From     common.Address  `json:"from" gencodec:"required"`
	GasPrice *hexutil.Big    `json:"gasPrice" gencodec:"required"`
	To       *common.Address `json:"to,omitempty"`
	Nonce    hexutil.Uint64  `json:"nonce" gencodec:"required"`
	Value    *hexutil.Big    `json:"value" gencodec:"required"`
	Gas      hexutil.Uint64  `json:"gas" gencodec:"required"`
	Input    hexutil.Bytes   `json:"input" gencodec:"required"`
}

type Dataset struct {
	db *gorocksdb.DB
}
type Options = struct {
	MaxOpenFiles int
}

var OptionsDefault = Options{
	MaxOpenFiles: 32,
}

func (self *Dataset) Init(path string, options *Options) (ret *Dataset, err error) {
	if options == nil {
		options = &OptionsDefault
	}
	db_opts := gorocksdb.NewDefaultOptions()
	db_opts.SetErrorIfExists(false)
	db_opts.SetMaxOpenFiles(options.MaxOpenFiles)
	self.db, err = gorocksdb.OpenDbForReadOnly(db_opts, path, false)
	db_opts.Destroy()
	ret = self
	return
}

func (self *Dataset) Close() {
	self.db.Close()
}

type Iterator struct {
	i *gorocksdb.Iterator
}

func (self *Iterator) HasNext() bool {
	return self.i != nil
}

func (self *Iterator) Next() (*Block, error) {
	if err := self.i.Err(); err != nil {
		return nil, err
	}
	v_s := self.i.Value()
	defer v_s.Free()
	ret := new(Block)
	err := json.Unmarshal(v_s.Data(), ret)
	if err == nil {
		self.i.Next()
	}
	return ret, err
}

func (self *Dataset) NewIterator(from types.BlockNum) (ret *Iterator) {
	ret = &Iterator{self.db.NewIterator(opts_r)}
	key := blk_num_to_key(from)
	ret.i.Seek(key)
	if !ret.i.Valid() {
		return
	}
	k_slice := ret.i.Key()
	defer k_slice.Free()
	if bytes.Compare(key, k_slice.Data()) != 0 {
		ret.i = nil
	}
	return
}

func (self *Iterator) Close() {
	if self.i != nil {
		self.i.Close()
	}
}

func (self *Dataset) GetBlock(n types.BlockNum) (*Block, error) {
	block_json, err := self.db.Get(opts_r, blk_num_to_key(n))
	if err != nil {
		return nil, err
	}
	defer block_json.Free()
	ret := new(Block)
	err = json.Unmarshal(block_json.Data(), ret)
	return ret, err
}

func blk_num_to_key(n types.BlockNum) []byte {
	return bin.BytesView(fmt.Sprintf("%09d", n))
}

var opts_r = gorocksdb.NewDefaultReadOptions()

func main() {
	dir, err_0 := os.UserHomeDir()
	util.PanicIfNotNil(err_0)
	ds, err := new(Dataset).Init(dir+"/blockchain", nil)
	util.PanicIfNotNil(err)
	fmt.Println(ds.GetBlock(10))
	fmt.Println(ds.GetBlock(0))
	for i := ds.NewIterator(0); i.HasNext(); {
		blk, err := i.Next()
		util.PanicIfNotNil(err)
		fmt.Println(blk.Number)
	}
}
