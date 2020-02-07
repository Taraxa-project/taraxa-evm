package rocksdb_ext

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/taraxa/awesomeProject1/util"
	"github.com/tecbot/gorocksdb"
	"strconv"
)

var Default_opts = gorocksdb.NewDefaultOptions()
var Default_opts_r = gorocksdb.NewDefaultReadOptions()
var Default_opts_w = gorocksdb.NewDefaultWriteOptions()

type RocksDBExt struct {
	*gorocksdb.DB
	cf      []*gorocksdb.ColumnFamilyHandle
	cf_opts []RocksDBExtColumnOpts
}
type RocksDBExtCFRWOpts = struct {
	CF_r_opts []*gorocksdb.ReadOptions
	CF_w_opts []*gorocksdb.WriteOptions
}

type RocksDBExtConfig = struct {
	RocksDBExtDBConfig
	ColumnOpts []RocksDBExtColumnOpts
}
type RocksDBExtDBConfig = struct {
	Path string
	Opts *gorocksdb.Options
}
type RocksDBExtColumnOpts = struct {
	Opts   *gorocksdb.Options
	Opts_r *gorocksdb.ReadOptions
	Opts_w *gorocksdb.WriteOptions
}

func NewRocksDBExt(cfg *RocksDBExtConfig) (self *RocksDBExt, err error) {
	if cfg.Opts == nil {
		cfg.Opts = gorocksdb.NewDefaultOptions()
	}
	cfg.Opts.SetCreateIfMissing(true)
	cfg.Opts.SetCreateIfMissingColumnFamilies(true)
	col_names := make([]string, len(cfg.ColumnOpts))
	col_names[0] = "default"
	for i := 1; i < len(col_names); i++ {
		col_names[i] = strconv.Itoa(i)
	}
	self = &RocksDBExt{cf_opts: cfg.ColumnOpts}
	cf_opts := make([]*gorocksdb.Options, len(col_names))
	for i := 0; i < len(cf_opts); i++ {
		col_opts := &cfg.ColumnOpts[i]
		if col_opts.Opts == nil {
			col_opts.Opts = gorocksdb.NewDefaultOptions()
		}
		if col_opts.Opts_r == nil {
			col_opts.Opts_r = gorocksdb.NewDefaultReadOptions()
		}
		if col_opts.Opts_w == nil {
			col_opts.Opts_w = gorocksdb.NewDefaultWriteOptions()
		}
		cf_opts[i] = col_opts.Opts
	}
	self.DB, self.cf, err = gorocksdb.OpenDbColumnFamilies(
		cfg.Opts,
		cfg.Path,
		col_names,
		cf_opts,
	)
	return
}

func (self *RocksDBExt) PutCol(col int, k, v []byte) error {
	return self.PutCF(self.cf_opts[col].Opts_w, self.cf[col], k, v)
}

func (self *RocksDBExt) BatchPutCol(batch *gorocksdb.WriteBatch, col int, k, v []byte) {
	batch.PutCF(self.cf[col], k, v)
}

func (self *RocksDBExt) GetCol(col int, k []byte) ([]byte, error) {
	slice, err := self.GetCF(self.cf_opts[col].Opts_r, self.cf[col], k)
	if err != nil {
		return nil, err
	}
	defer slice.Free()
	return slice.Data(), nil
}

func (self *RocksDBExt) Find(col int, key []byte, floor bool) (k, v []byte, err error) {
	i := self.NewIteratorCF(self.cf_opts[col].Opts_r, self.cf[col])
	if err = i.Err(); err != nil {
		return
	}
	if floor {
		i.SeekForPrev(key)
	} else {
		i.Seek(key)
	}
	if err = i.Err(); err != nil || !i.Valid() {
		return
	}
	k_slice, v_slice := i.Key(), i.Value()
	defer k_slice.Free()
	defer v_slice.Free()
	return k_slice.Data(), v_slice.Data(), nil
}

func (self *RocksDBExt) Dump() {
	for col_num, col := range self.cf {
		i := self.NewIteratorCF(Default_opts_r, col)
		for i.SeekToFirst(); i.Valid(); i.Next() {
			util.PanicIfNotNil(i.Err())
			k_slice, v_slice := i.Key(), i.Value()
			defer k_slice.Free()
			defer v_slice.Free()
			fmt.Println(
				"Column:", strconv.Itoa(col_num),
				"|| Key:", util.BytesToStrPadded(k_slice.Data()),
				"|| Value:", util.BytesToStrPadded(v_slice.Data()))
		}
	}
}
