package main

import (
	"github.com/Taraxa-project/taraxa-evm/ethdb"
	eth_ethdb "github.com/ethereum/go-ethereum/ethdb"
)

type dbAdapter struct {
	ethdb.MutableTransactionalDatabase
}

func (this *dbAdapter) Has(key []byte) (bool, error) {
	panic("implement me")
}

func (this *dbAdapter) NewBatch() eth_ethdb.Batch {
	panic("implement me")
}

func (this *dbAdapter) Close() error {
	panic("implement me")
}

func (this *dbAdapter) HasAncient(kind string, number uint64) (bool, error) {
	panic("implement me")
}

func (this *dbAdapter) Ancient(kind string, number uint64) ([]byte, error) {
	panic("implement me")
}

func (this *dbAdapter) Ancients() (uint64, error) {
	panic("implement me")
}

func (this *dbAdapter) AncientSize(kind string) (uint64, error) {
	panic("implement me")
}

func (this *dbAdapter) AppendAncient(number uint64, hash, header, body, receipt, td []byte) error {
	panic("implement me")
}

func (this *dbAdapter) TruncateAncients(n uint64) error {
	panic("implement me")
}

func (this *dbAdapter) Sync() error {
	panic("implement me")
}

func (this *dbAdapter) NewIterator() eth_ethdb.Iterator {
	panic("implement me")
}

func (this *dbAdapter) NewIteratorWithStart(start []byte) eth_ethdb.Iterator {
	panic("implement me")
}

func (this *dbAdapter) NewIteratorWithPrefix(prefix []byte) eth_ethdb.Iterator {
	panic("implement me")
}

func (this *dbAdapter) Stat(property string) (string, error) {
	panic("implement me")
}

func (this *dbAdapter) Compact(start []byte, limit []byte) error {
	panic("implement me")
}
