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

package vm

import (
	"github.com/Taraxa-project/taraxa-evm/taraxa/util"
)

// TODO compress to error codes
// List execution errors
var (
	ErrOutOfGas                       = util.ErrorString("out of gas")
	ErrIntrinsicGas                   = util.ErrorString("intrinsic gas too low")
	ErrCodeStoreOutOfGas              = util.ErrorString("contract creation code storage out of gas")
	ErrDepth                          = util.ErrorString("max call depth exceeded")
	ErrInsufficientBalanceForTransfer = util.ErrorString("insufficient balance for transfer")
	ErrContractAddressCollision       = util.ErrorString("contract address collision")
	ErrInsufficientBalanceForGas      = util.ErrorString("insufficient balance to pay for gas")
	ErrNonceTooHigh                   = util.ErrorString("nonce too high")
	ErrNonceTooLow                    = util.ErrorString("nonce too low")
	ErrWriteProtection                = util.ErrorString("write protection")
)
