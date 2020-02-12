package rocksdb_ext

import (
	"bytes"
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
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

	cache        map[int]map[string][]byte
	writes       uint64
	reads        uint64
	dont_profile bool
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

// TODO deallocate rocksdb objects
func NewRocksDBExt(cfg *RocksDBExtConfig) (self *RocksDBExt, err error) {
	if cfg.Opts == nil {
		cfg.Opts = gorocksdb.NewDefaultOptions()
	}
	cfg.Opts.SetCreateIfMissing(true)
	cfg.Opts.SetCreateIfMissingColumnFamilies(true)
	if len(cfg.ColumnOpts) == 0 {
		cfg.ColumnOpts = append(cfg.ColumnOpts, RocksDBExtColumnOpts{})
	}
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

	self.cache = make(map[int]map[string][]byte)
	for col := range cfg.ColumnOpts {
		self.cache[col] = make(map[string][]byte)
	}
	return
}

func (self *RocksDBExt) PutCol(col int, k, v []byte) error {
	if !self.dont_profile {
		self.writes++
	}
	self.cache[col][string(k)] = v
	return nil
	//return self.PutCF(self.cf_opts[col].Opts_w, self.cf[col], k, v)
}

func (self *RocksDBExt) BatchPutCol(batch *gorocksdb.WriteBatch, col int, k, v []byte) {
	if !self.dont_profile {
		self.writes++
	}
	self.cache[col][string(k)] = v
	//batch.PutCF(self.cf[col], k, v)
}

func (self *RocksDBExt) ToggleProfiling() {
	self.dont_profile = !self.dont_profile
}

func (self *RocksDBExt) GetCol(col int, k []byte) ([]byte, error) {
	if !self.dont_profile {
		self.reads++
	}
	k_str := string(k)
	ret, ok := self.cache[col][k_str]
	if !ok {
		//slice, err := self.GetCF(self.cf_opts[col].Opts_r, self.cf[col], k)
		//if err != nil {
		//	return nil, err
		//}
		//defer slice.Free()
		//ret = common.CopyBytes(slice.Data())
		//self.cache[col][k_str] = ret
	}
	return ret, nil
}

func (self *RocksDBExt) Commit(batch *gorocksdb.WriteBatch) error {
	//if !self.dont_profile {
	//	fmt.Println("reads:", self.reads)
	//	fmt.Println("writes:", self.writes)
	//}
	self.reads = 0
	self.writes = 0
	//for col := range self.cache {
	//	self.cache[col] = make(map[string][]byte)
	//}
	return nil
	//start := time.Now()
	//defer func() {
	//	fmt.Println("commit took (sec)", float64(time.Now().Sub(start))/float64(time.Second))
	//}()
	//return self.Write(Default_opts_w, batch)
}

func (self *RocksDBExt) Find(col int, key []byte, floor bool) (k, v []byte, err error) {
	//panic("foo")
	r_opts := gorocksdb.NewDefaultReadOptions()
	//r_opts.SetIterateUpperBound(key)
	r_opts.SetTailing(true)
	r_opts.SetReadaheadSize((1 << (2 * 9)) * 2)
	i := self.NewIteratorCF(self.cf_opts[col].Opts_r, self.cf[col])
	//i := self.NewIteratorCF(self.cf_opts[col].Opts_r, self.cf[col])
	defer i.Close()
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
	return common.CopyBytes(k_slice.Data()), common.CopyBytes(v_slice.Data()), nil
}

func (self *RocksDBExt) MaxForPrefix(col int, key []byte, prefix_len int) (ret []byte, err error) {
	found_k, found_v, err_0 := self.Find(col, key, true)
	if err = err_0; err_0 == nil && bytes.HasPrefix(found_k, key[:prefix_len]) {
		ret = found_v
	}
	return
}

func (self *RocksDBExt) Dump() {
	fmt.Println("Dumping db...")
	for col_num, col := range self.cache {
		for k, v := range col {
			fmt.Println(
				"Column:", strconv.Itoa(col_num),
				"|| Key:", util.BytesToStrPadded([]byte(k)),
				"|| Value:", util.BytesToStrPadded(v))
		}
	}
	//for col_num, col := range self.cf {
	//	i := self.NewIteratorCF(Default_opts_r, col)
	//	for i.SeekToFirst(); i.Valid(); i.Next() {
	//		util.PanicIfNotNil(i.Err())
	//		k_slice, v_slice := i.Key(), i.Value()
	//		defer k_slice.Free()
	//		defer v_slice.Free()
	//		fmt.Println(
	//			"Column:", strconv.Itoa(col_num),
	//			"|| Key:", util.BytesToStrPadded(k_slice.Data()),
	//			"|| Value:", util.BytesToStrPadded(v_slice.Data()))
	//	}
	//}
}
