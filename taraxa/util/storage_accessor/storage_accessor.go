package storage_accessor

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/taraxa/util/keccak256"
)

type StorageAccessor struct {
	key common.Hash
}

func (s *StorageAccessor) reset() *StorageAccessor {
	s.key = common.ZeroHash
	return s
}

func (s *StorageAccessor) SetPos(k *common.Hash) *StorageAccessor {
	s.key = *k
	return s
}

func (s *StorageAccessor) Key() common.Hash {
	return s.key
}

func (s *StorageAccessor) MapAtHash(k common.Hash) *StorageAccessor {
	c := new(StorageAccessor)
	c.key = *keccak256.Hash(k.Bytes(), s.key.Bytes())
	return c
}

func (s *StorageAccessor) MapAt(k int64) *StorageAccessor {
	return s.MapAtHash(common.BytesToHash(big.NewInt(k).Bytes()))
}

func (s *StorageAccessor) Array() *StorageAccessor {
	c := new(StorageAccessor)
	c.key = *keccak256.Hash(s.key.Bytes())
	return c
}

func (s *StorageAccessor) AtBig(i *big.Int) *StorageAccessor {
	c := new(StorageAccessor)
	c.key = *s.key.Add(i)
	return c
}

func (s *StorageAccessor) At(i int) *StorageAccessor {
	return s.AtBig(big.NewInt(int64(i)))
}

func (s *StorageAccessor) Field(i int) *StorageAccessor {
	return s.AtBig(big.NewInt(int64(i)))
}

func (s *StorageAccessor) ArraySize() common.Hash {
	return s.key
}

func (s *StorageAccessor) Struct() *StorageAccessor {
	return s
}
