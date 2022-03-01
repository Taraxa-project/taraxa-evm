// Copyright 2016 The go-ethereum Authors
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

package vm

import (
	"math/big"

	"github.com/Taraxa-project/taraxa-evm/common"
)

// State is an EVM database for full state querying.
type State interface {
	GetAccount(*common.Address) StateAccount
	AddLog(LogRecord)
	GetLogs() []LogRecord
	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64
	RevertToSnapshot(int)
	Snapshot() int
}

type StateAccount interface {
	Address() *common.Address
	GetBalance() *big.Int
	GetNonce() *big.Int
	GetCodeHash() *common.Hash
	GetCode() []byte
	GetCodeSize() uint64
	GetCommittedState(*big.Int) *big.Int
	GetState(*big.Int) *big.Int
	HasSuicided() bool
	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	IsNotNIL() bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	IsEIP161Empty() bool
	SetCode([]byte)
	AddBalance(*big.Int)
	SubBalance(*big.Int)
	IncrementNonce()
	SetState(*big.Int, *big.Int)
	Suicide(*common.Address)
}

func BalanceGTE(acc StateAccount, val *big.Int) bool {
	return val.Sign() == 0 || acc.GetBalance().Cmp(val) >= 0
}
