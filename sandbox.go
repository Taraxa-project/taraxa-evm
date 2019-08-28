package main

import (
	"github.com/Taraxa-project/taraxa-evm/common"
	"github.com/Taraxa-project/taraxa-evm/misc"
)

type AccountKey = [common.AddressLength]byte
type AccountStorageKey = [common.AddressLength + common.HashLength]byte
type AccountFieldMask byte
type AccountFieldKey = [common.AddressLength + 1]byte

const (
	balance AccountFieldMask = iota
	nonce
	codeHash
	storageRoot
	code
)

func accountFieldKey(addr *common.Address, mask AccountFieldMask) *AccountFieldKey {
	ret := new(AccountFieldKey)
	copy(ret[:], addr[:])
	ret[common.AddressLength] = byte(mask)
	return ret
}

func accountStorageKey(addr *common.Address, location *common.Hash) *AccountStorageKey {
	ret := new(AccountStorageKey)
	copy(ret[:], addr[:])
	copy(ret[common.AddressLength:], location[:])
	return ret
}

func main() {
	misc.DumpStateRocksdb(
		"/workspace/data/eth_mainnet_rocksdb/state",
		"0x03c6d7095795ac2fa6fc6fc3a83c6c9cb085b7637ab2fa31b228f3d7bfeaac55")
	//db, err := gorocksdb.OpenDbForReadOnly(
	//	gorocksdb.NewDefaultOptions(), "/workspace/data/ethereum_blockchain_mainnet_rocksdb", false)
	//util.PanicIfPresent(err)
	//itr := db.NewIterator(gorocksdb.NewDefaultReadOptions())
	//fmt.Println("foo")
	//for ; itr.Valid(); itr.Next() {
	//	fmt.Println("foo")
	//	_k, _v := itr.Key(), itr.Value()
	//	k, v := string(_k.Data()), string(_v.Data())
	//	fmt.Printf("Key: %s Value: %s\n", k, v)
	//	_k.Free()
	//	_v.Free()
	//}
}
