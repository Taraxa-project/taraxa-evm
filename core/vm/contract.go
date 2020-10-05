// Copyright 2015 The go-ethereum Authors
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

// Contract represents an ethereum contract in the state database. It contains
// the contract code, calling arguments.
type Contract struct {
	CallFrame
	code                   CodeAndHash
	code_jumpdest_analysis bitvec
}
type CallFrame = struct {
	// CallerAddress is the result of the caller which initialised this
	// contract. However when the "call method" is delegated this value
	// needs to be initialised to that of the caller's caller.
	CallerAccount StateAccount
	Account       StateAccount
	Input         []byte
	Gas           uint64
	Value         *big.Int
}

type CodeAndHash struct {
	Code     []byte
	CodeHash *common.Hash
}

// NewContract returns a new contract environment for the execution of EVM.
func NewContract(frame CallFrame, code CodeAndHash) (ret Contract) {
	ret.CallFrame = frame
	ret.code = code
	return
}

func (self *Contract) GetCode() []byte {
	return self.code.Code
}

// TODO optimize and refactor
func (self *Contract) ValidJumpdest(evm *EVM, dest *big.Int) bool {
	// PC cannot go beyond len(code) and certainly can't be bigger than 63bits.
	// Don't bother checking for JUMPDEST in that case.
	if !dest.IsUint64() {
		return false
	}
	udest := dest.Uint64()
	if self.GetOp(udest) != JUMPDEST {
		return false
	}
	analysis := self.code_jumpdest_analysis
	if cached := analysis != nil; !cached {
		if analysis, cached = evm.analyze_jumpdests(self.code); !cached {
			self.code_jumpdest_analysis = analysis
		}
	}
	return analysis.codeSegment(udest)
}

// GetOp returns the n'th element in the contract's byte array
func (self *Contract) GetOp(n uint64) OpCode {
	if n < uint64(len(self.code.Code)) {
		return OpCode(self.code.Code[n])
	}
	return 0
}

// UseGas attempts the use gas and subtracts it and returns true on success
func (self *Contract) UseGas(gas uint64) (ok bool) {
	if self.Gas < gas {
		return false
	}
	self.Gas -= gas
	return true
}
