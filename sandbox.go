package main

import (
	"fmt"
	"github.com/Taraxa-project/taraxa-evm/common"
	"math/big"
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
	addr := common.Address{}
	h := common.BigToHash(new(big.Int).SetUint64(3234342422342443423))
	var f1, f2 interface{} = accountStorageKey(&addr, &h), accountStorageKey(&addr, &h)
	f3 := f1.(*AccountStorageKey)
	m := make(map[interface{}]interface{})
	m[f1] = 1
	m[f2] = 1
	m[f3] = 1
	m[*f3] = 1
	m[new(int)] = 1
	fmt.Println(f1 == f2)
	fmt.Println(m[f1] == m[f2] && m[f2] == m[f3] && m[f3] == m[*f3])
	//fmt.Println(ff)
}
