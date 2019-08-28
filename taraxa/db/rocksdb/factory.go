package rocksdb

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/tecbot/gorocksdb"
)

type Factory struct {
	File                      string  `json:"file"`
	ReadOnly                  bool    `json:"readOnly"`
	ErrorIfExists             bool    `json:"errorIfExists"`
	DontCreateIfMissing       bool    `json:"dontCreateIfMissing"`
	MaxOpenFiles              int     `json:"maxOpenFiles"`
	BloomFilterCapacity       int     `json:"bloomFilterCapacity"`
	BlockCacheSize            uint64  `json:"blockCacheSize"`
	WriteBufferSize           int     `json:"writeBufferSize"`
	Parallelism               int     `json:"parallelism"`
	OptimizeForPointLookup    *uint64 `json:"optimizeForPointLookup"`
	MaxFileOpeningThreads     *int    `json:"maxFileOpeningThreads"`
	//TODO CacheIndexAndFilterBlocks *bool   `json:"cacheIndexAndFilterBlocks"`
}

func (this *Factory) NewInstance() (ret ethdb.MutableTransactionalDatabase, err error) {
	opts := gorocksdb.NewDefaultOptions()
	blockOpts := gorocksdb.NewDefaultBlockBasedTableOptions()
	bloomFilter := gorocksdb.NewBloomFilter(util.Max(10, this.BloomFilterCapacity))
	blockOpts.SetFilterPolicy(bloomFilter)
	if this.BlockCacheSize > 0 {
		blockOpts.SetBlockCache(gorocksdb.NewLRUCache(this.BlockCacheSize))
	}
	opts.SetCreateIfMissing(!this.DontCreateIfMissing)
	opts.SetBlockBasedTableFactory(blockOpts)
	if this.WriteBufferSize > 0 {
		opts.SetWriteBufferSize(this.WriteBufferSize)
	}
	if this.MaxOpenFiles > 0 {
		opts.SetMaxOpenFiles(this.MaxOpenFiles)
	}
	if this.Parallelism > 0 {
		opts.IncreaseParallelism(this.Parallelism)
	}
	if this.OptimizeForPointLookup != nil {
		opts.SetAllowConcurrentMemtableWrites(false)
		opts.OptimizeForPointLookup(*this.OptimizeForPointLookup)
	}
	opts.SetErrorIfExists(this.ErrorIfExists)
	database := new(Database)
	ret = database
	database.writeOpts = gorocksdb.NewDefaultWriteOptions()
	database.readOpts = gorocksdb.NewDefaultReadOptions()
	if this.ReadOnly {
		database.db, err = gorocksdb.OpenDbForReadOnly(opts, this.File, this.ErrorIfExists)
	} else {
		database.db, err = gorocksdb.OpenDb(opts, this.File)
	}
	return
}
