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
	GetBalance(common.Address) *big.Int
	HasBalance(common.Address) bool
	AssertBalanceGTE(common.Address, *big.Int) bool
	GetNonce(common.Address) uint64
	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	GetCodeSize(common.Address) uint64
	GetCommittedState(common.Address, common.Hash) *big.Int
	GetState(common.Address, common.Hash) *big.Int
	HasSuicided(common.Address) bool
	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	SetCode(common.Address, []byte)
	AddBalance(common.Address, *big.Int)
	SubBalance(common.Address, *big.Int)
	IncrementNonce(common.Address)
	SetState(common.Address, common.Hash, *big.Int)
	Suicide(addr, newAddr common.Address)

	AddLog(LogRecord)
	GetLogs() []LogRecord

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	RevertToSnapshot(int)
	Snapshot() int
}
