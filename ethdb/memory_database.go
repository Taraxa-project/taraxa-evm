// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethdb

import (
	"errors"
	"sync"

	"github.com/Taraxa-project/taraxa-evm/common"
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte),
	}
}

func NewMemDatabaseWithCap(size int) *MemDatabase {
	return &MemDatabase{
		db: make(map[string][]byte, size),
	}
}

func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

func (db *MemDatabase) Close() {}

type mapWriter struct {
	db *MemDatabase
}

func (w *mapWriter) Put(key []byte, value []byte) error {
	w.db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (w *mapWriter) Delete(key []byte) error {
	delete(w.db.db, string(key))
	return nil
}

func (db *MemDatabase) NewBatch() Batch {
	writer := &mapWriter{db: db}
	return &MemBatch{
		writer:     writer,
		commitLock: &db.lock,
	}
}

func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct {
	k, v []byte
	del  bool
}

type putterAndDeleter interface {
	Putter
	Deleter
}

type MemBatch struct {
	writer     putterAndDeleter
	commitLock *sync.RWMutex
	writes     []kv
	size       int
}

func (b *MemBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

func (b *MemBatch) Delete(key []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), nil, true})
	b.size += 1
	return nil
}

func (b *MemBatch) Write() (err error) {
	if b.commitLock != nil {
		b.commitLock.Lock()
		defer b.commitLock.Unlock()
	}
	for _, kv := range b.writes {
		if kv.del {
			err = b.writer.Delete(kv.k)
			if err != nil {
				return
			}
			continue
		}
		err = b.writer.Put(kv.k, kv.v)
		if err != nil {
			return
		}
	}
	return nil
}

func (b *MemBatch) ValueSize() int {
	return b.size
}

func (b *MemBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}
