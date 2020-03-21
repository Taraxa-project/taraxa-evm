package rocksdb

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
	"github.com/tecbot/gorocksdb"
)

type Database struct {
	writeOpts *gorocksdb.WriteOptions
	readOpts  *gorocksdb.ReadOptions
	db        *gorocksdb.DB
	tasks     chan func()
}
type Config struct {
	File                          string `json:"file"`
	ReadOnly                      bool   `json:"readOnly"`
	ErrorIfExists                 bool   `json:"errorIfExists"`
	DontCreateIfMissing           bool   `json:"dontCreateIfMissing"`
	MaxOpenFiles                  int    `json:"maxOpenFiles"`
	BloomFilterCapacity           int    `json:"bloomFilterCapacity"`
	BlockCacheSize                uint64 `json:"blockCacheSize"`
	WriteBufferSize               int    `json:"writeBufferSize"`
	Parallelism                   int    `json:"parallelism"`
	MaxBackgroundFlushes          int    `json:"maxBackgroundFlushes"`
	OptimizeForPointLookup        uint64 `json:"optimizeForPointLookup"`
	MaxFileOpeningThreads         int    `json:"maxFileOpeningThreads"`
	UseDirectReads                bool   `json:"useDirectReads"`
	UseDirectWrites               bool   `json:"useDirectWrites"`
	AllowMmapReads                bool   `json:"allowMmapReads"`
	TargetFileSizeBase            uint64 `json:"targetFileSizeBase"`
	TargetFileSizeMultiplier      int    `json:"targetFileSizeMultiplier"`
	LevelCompactionMemtableBudget uint64 `json:"levelCompactionMemtableBudget"`
	//TODO CacheIndexAndFilterBlocks *bool   `json:"cacheIndexAndFilterBlocks"`
}

func New(cfg *Config) *Database {
	opts := gorocksdb.NewDefaultOptions()
	if cfg.OptimizeForPointLookup != 0 {
		opts.SetAllowConcurrentMemtableWrites(false)
		opts.OptimizeForPointLookup(cfg.OptimizeForPointLookup)
	} else {
		blockOpts := gorocksdb.NewDefaultBlockBasedTableOptions()
		bloomFilter := gorocksdb.NewBloomFilter(util.Max(10, cfg.BloomFilterCapacity))
		blockOpts.SetFilterPolicy(bloomFilter)
		if cfg.BlockCacheSize != 0 {
			blockOpts.SetBlockCache(gorocksdb.NewLRUCache(cfg.BlockCacheSize))
		}
		opts.SetBlockBasedTableFactory(blockOpts)
	}
	if cfg.LevelCompactionMemtableBudget != 0 {
		opts.OptimizeLevelStyleCompaction(cfg.LevelCompactionMemtableBudget)
	}
	if cfg.TargetFileSizeBase != 0 {
		opts.SetTargetFileSizeBase(cfg.TargetFileSizeBase)
	}
	if cfg.TargetFileSizeMultiplier != 0 {
		opts.SetTargetFileSizeMultiplier(cfg.TargetFileSizeMultiplier)
	}
	if cfg.WriteBufferSize != 0 {
		opts.SetWriteBufferSize(cfg.WriteBufferSize)
	}
	if cfg.MaxOpenFiles != 0 {
		opts.SetMaxOpenFiles(cfg.MaxOpenFiles)
	}
	if cfg.Parallelism != 0 {
		opts.IncreaseParallelism(cfg.Parallelism)
	}
	if cfg.MaxFileOpeningThreads != 0 {
		opts.SetMaxFileOpeningThreads(cfg.MaxFileOpeningThreads)
	}
	if cfg.MaxBackgroundFlushes != 0 {
		opts.SetMaxBackgroundFlushes(cfg.MaxBackgroundFlushes)
	}
	opts.SetUseDirectIOForFlushAndCompaction(cfg.UseDirectWrites)
	opts.SetUseDirectReads(cfg.UseDirectReads)
	opts.SetAllowMmapReads(cfg.AllowMmapReads)
	opts.SetErrorIfExists(cfg.ErrorIfExists)
	opts.SetCreateIfMissing(!cfg.DontCreateIfMissing)
	ret, err := new(Database), error(nil)
	ret.writeOpts = gorocksdb.NewDefaultWriteOptions()
	ret.readOpts = gorocksdb.NewDefaultReadOptions()
	ret.readOpts.SetVerifyChecksums(false)
	if cfg.ReadOnly {
		ret.db, err = gorocksdb.OpenDbForReadOnly(opts, cfg.File, cfg.ErrorIfExists)
	} else {
		ret.db, err = gorocksdb.OpenDb(opts, cfg.File)
	}
	util.PanicIfNotNil(err)
	tasks := make(chan func(), 2048)
	go func() {
		for {
			if t, ok := <-tasks; ok {
				t()
				continue
			}
			return
		}
	}()
	ret.tasks = tasks
	return ret
}

func (self *Database) Unwrap() *gorocksdb.DB {
	return self.db
}

func (self *Database) Put(key []byte, value []byte) error {
	return self.db.Put(self.writeOpts, key, value)
}

func (self *Database) Get(key []byte) ([]byte, error) {
	val_handle, err := self.db.GetPinned(self.readOpts, key)
	if err != nil {
		return nil, err
	}
	ret := common.CopyBytes(val_handle.Data())
	self.tasks <- val_handle.Destroy
	return ret, nil
}

func (self *Database) Close() {
	self.tasks <- func() {
		self.readOpts.Destroy()
		self.writeOpts.Destroy()
		self.db.Close()
		close(self.tasks)
	}
	*self = Database{}
}

func (self *Database) NewBatch() ethdb.Batch {
	return &batch{
		db:    self,
		batch: gorocksdb.NewWriteBatch(),
	}
}
